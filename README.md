# Supply Chain Security Gateway

A reference architecture and **<ins>proof of concept implementation</ins>** of a supply chain security gateway with the goal of enforcing sane security policies to an organization's consumption of 3rd party software (dependencies) in its own products.

- [Supply Chain Security Gateway](#supply-chain-security-gateway)
  - [TL;DR](#tldr)
  - [Architecture](#architecture)
    - [Data Plane Flow](#data-plane-flow)
  - [Usage](#usage)
    - [Configuring Upstream and Routes](#configuring-upstream-and-routes)
    - [Configuring Environments](#configuring-environments)
    - [Authentication](#authentication)
      - [Ingress Authentication](#ingress-authentication)
        - [Basic Authentication](#basic-authentication)
  - [Development](#development)
    - [PDP Development](#pdp-development)
    - [Policy Development](#policy-development)
    - [Tap Development](#tap-development)
    - [Debug NATS Messaging](#debug-nats-messaging)
  - [Contribution](#contribution)

## TL;DR

Ensure git submodules are updated locally

```bash
git submodule update --init --recursive
```

Initialize keys and certificates for mTLS

```bash
./bootstrap.sh
```

> This will generate root certificate, per service certificates in `pki/`.

[TLS SAN](https://en.wikipedia.org/wiki/Subject_Alternative_Name) must be correctly set in the generated certificate for the mTLS to work correctly. Verify using:

```bash
openssl x509 -noout -text \
  -in ./pki/nats-server/server.crt | grep "DNS:nats-server"
```

Start the services using `docker-compose`

```bash
docker-compose up -d
```

Verify all the services are active

```bash
docker-compose ps
```

Use the gateway using a `demo-client`

```bash
cd demo-clients/java-gradle && ./gradlew assemble --refresh-dependencies
```

At this point, you should see logs generated by gateway and the policy decision service and multiple artefacts that are violating configured policy are blocked by the gateway

```bash
docker-compose logs envoy
docker-compose logs pdp
```

The `gradle` build should fail with an error message indicating a dependency was blocked by the gateway.

```
> Could not resolve all files for configuration ':app:compileClasspath'.
   > Could not resolve org.apache.logging.log4j:log4j:2.16.0.
     Required by:
         project :app
      > Could not resolve org.apache.logging.log4j:log4j:2.16.0.
         > Could not get resource 'http://localhost:10000/maven2/org/apache/logging/log4j/log4j/2.16.0/log4j-2.16.0.pom'.
            > Could not GET 'http://localhost:10000/maven2/org/apache/logging/log4j/log4j/2.16.0/log4j-2.16.0.pom'. Received status code 403 from server: Forbidden
```

> Refer to `policies/example.rego` for the policy that blocked this artefact

Edit `config/global.yml` and set `pdp.monitor_mode=true` to enable only monitoring and disable policy enforcement. Restart the containers for the changes to take effect.

```bash
docker-compose up --force-recreate --remove-orphans --build -d
```

Run the build again to see it compile successfully.

```bash
cd demo-clients/java-gradle && ./gradlew build --refresh-dependencies
```

## Architecture

![HLD](docs/images/supply-chain-gateway-hld.png)

### Data Plane Flow

![Data Plane Flow](docs/images/data-plane-flow.png)

## Usage

If you are developing on any of the service and want to force re-create the containers with updated image:

```bash
docker-compose up --force-recreate --remove-orphans --build -d
```

### Configuring Upstream and Routes

The configuration plane is currently half baked. It needs a tool and a single source of truth to generate configuration for Envoy and Gateway. For now, look at:

1. `config/global.yml`
2. `config/envoy.yml`

> The route definitions in `envoy.yml` must match the path patterns in `global.yml`

### Configuring Environments

To use the gateway in a CI or developer local environment, package managers need to be configured to use the gateway URL and credentials as repository.

[PacMan](pacman/README.md) makes it easy to automatically configure an environment to use the gateway for downloading 3rd party dependencies.

### Authentication

There are two authentication points:

1. Ingress
2. Egress

Ingress authentication is for incoming requests to the gateway and can be used to identify who is accessing the gateway.

Egress authentication is for upstream repositories, especially the ones that need authentication e.g. CodeArtifact, JFrog, Nexus etc.

- [Ingress Gateway Authentication](docs/Gateway-Authentication.md)

#### Ingress Authentication

##### Basic Authentication

Use `htpasswd` to add users:

```bash
htpasswd -nbB user1 password1 >> ./config/gateway-auth-basic
```

Enable authentication for upstream in `config/global.yml`

## Development

### PDP Development

Build and run the PDP using:

```bash
cd services && make
GLOBAL_CONFIG_PATH=../config/global.yml PDP_POLICY_PATH=../policies ./out/pdp-server
```

PDP listens on `0.0.0.0:9000`. To use the host instance of PDP, edit `config/envoy.yml` and set the address of the `ExtAuthZ` plugin to your host network address.

### Policy Development

Policies are written in [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) and evaluated with [Open Policy Agent](https://www.openpolicyagent.org/docs/latest/integration/#integrating-with-the-go-api)

To run policy test cases:

```bash
cd policies && make test
```

* Refer to `policies/example.rego` for policy example
* Policies are load from `./policies` directory

### Tap Development

The *Tap Service* is integrated as a Envoy [ExtProc](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_proc_filter) filter. This means, it has greater control over Envoy's request processing life-cycle and can make changes if required.

Currently, it is used for publishing events for data collection only but in future may be extended to support other use-cases. Tap service internally implements a handler chain to delegate an Envoy event to its internal handlers. Example:

```go
tapService, err := tap.NewTapService(config, []tap.TapHandlerRegistration{
  tap.NewTapEventPublisherRegistration(),
})
```

To build and use from host:

```bash
cd services && make
GLOBAL_CONFIG_PATH=../config/global.yml ./out/tap-server
```

> To use Tap service from host, edit `envoy.yml` and change address of `ext-proc-tap` cluster.

### Debug NATS Messaging

Start a docker container with `nats` client

```bash
docker run --rm -it \
   --network supply-chain-security-gateway_default \
   -v `pwd`:/workspace \
   synadia/nats-box
```

Subscribe to a subject and receive messages

```bash
GODEBUG=x509ignoreCN=0 nats sub \
   --tlscert=/workspace/pki/tap/server.crt \
   --tlskey=/workspace/pki/tap/server.key \
   --tlsca=/workspace/pki/root.crt \
   --server=tls://nats-server:4222 \
   com.msg.event.upstream.request
```

## Contribution

Look at [Github issues](https://github.com/abhisek/supply-chain-security-gateway/issues)
