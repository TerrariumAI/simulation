## Simulation Service

This repo holds the code for the Terrarium.ai enivronment, collective, and all other utility packages.

## Running locally (without kubernetes)

If you just want to run the services locally, you first need to install Go and run a local instance of redis on your computer

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
