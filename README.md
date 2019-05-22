## Simulation Service

This is the code that runs the AI environment simulation. You can also use this to train your AI by running the simulation locally and connecting directly to it via Python.

NOTE: If you are trying to run the Web-Client and connect to this, you will first need to start up an Envoy-Proxy service and connect to that in order to translate the messages.

## Flags

**-grpc-port=<PORT_NUMBER>** The port the gRPC server will run on  
**-http-port=<PORT_NUMBER>** The port the REST server will run on  
**-log-level=<LEVEL>** The amount of logging you want
**-env=<ENVIRONMENT>** The env can either be "prod", "training", or "testing".

## Firebase Credentials

When running the Simulation service, you need Firebase credentials in order to connect to a database.

### Testing

`-env=testing`  
The testing environment runs completely offline. This is used for unit testing.

### Staging

`-env=staging`  
The staging environment runs using our staging Firebase servers and a local Redis server.

### Prod

`-env=prod`  
Looks for a file called `serviceAccountKey.json` in the root directory to use the prod environment.

### Getting Firebase Creds

If you are trying to run this service locally, you can go [here](https://firebase.google.com/docs/admin/setup) to get a tutorial on generating these keys under the "Add Firebase to your app" section.

### Proposed method of storing in Redis

Create an entity at x=079, y=209 with an id=0
`ZADD entities 0 027099:0`

Add data to the entity
`HSET entities.content:0 class agent`

Query for the entity by position
