/**
 * @fileoverview gRPC-Web generated client stub for v1
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!



const grpc = {};
grpc.web = require('grpc-web');


var google_api_annotations_pb = require('./google/api/annotations_pb.js')

var protoc$gen$swagger_options_annotations_pb = require('./protoc-gen-swagger/options/annotations_pb.js')
const proto = {};
proto.v1 = require('./simulation-service_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.v1.SimulationServiceClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

  /**
   * @private @const {?Object} The credentials to be used to connect
   *    to the server
   */
  this.credentials_ = credentials;

  /**
   * @private @const {?Object} Options for the client
   */
  this.options_ = options;
};


/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.v1.SimulationServicePromiseClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!proto.v1.SimulationServiceClient} The delegate callback based client
   */
  this.delegateClient_ = new proto.v1.SimulationServiceClient(
      hostname, credentials, options);

};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.v1.CreateAgentRequest,
 *   !proto.v1.CreateAgentResponse>}
 */
const methodInfo_SimulationService_CreateAgent = new grpc.web.AbstractClientBase.MethodInfo(
  proto.v1.CreateAgentResponse,
  /** @param {!proto.v1.CreateAgentRequest} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.v1.CreateAgentResponse.deserializeBinary
);


/**
 * @param {!proto.v1.CreateAgentRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.v1.CreateAgentResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.v1.CreateAgentResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServiceClient.prototype.createAgent =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/v1.SimulationService/CreateAgent',
      request,
      metadata,
      methodInfo_SimulationService_CreateAgent,
      callback);
};


/**
 * @param {!proto.v1.CreateAgentRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.v1.CreateAgentResponse>}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServicePromiseClient.prototype.createAgent =
    function(request, metadata) {
  return new Promise((resolve, reject) => {
    this.delegateClient_.createAgent(
      request, metadata, (error, response) => {
        error ? reject(error) : resolve(response);
      });
  });
};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.v1.GetAgentRequest,
 *   !proto.v1.GetAgentResponse>}
 */
const methodInfo_SimulationService_GetAgent = new grpc.web.AbstractClientBase.MethodInfo(
  proto.v1.GetAgentResponse,
  /** @param {!proto.v1.GetAgentRequest} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.v1.GetAgentResponse.deserializeBinary
);


/**
 * @param {!proto.v1.GetAgentRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.v1.GetAgentResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.v1.GetAgentResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServiceClient.prototype.getAgent =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/v1.SimulationService/GetAgent',
      request,
      metadata,
      methodInfo_SimulationService_GetAgent,
      callback);
};


/**
 * @param {!proto.v1.GetAgentRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.v1.GetAgentResponse>}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServicePromiseClient.prototype.getAgent =
    function(request, metadata) {
  return new Promise((resolve, reject) => {
    this.delegateClient_.getAgent(
      request, metadata, (error, response) => {
        error ? reject(error) : resolve(response);
      });
  });
};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.v1.DeleteAgentRequest,
 *   !proto.v1.DeleteAgentResponse>}
 */
const methodInfo_SimulationService_DeleteAgent = new grpc.web.AbstractClientBase.MethodInfo(
  proto.v1.DeleteAgentResponse,
  /** @param {!proto.v1.DeleteAgentRequest} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.v1.DeleteAgentResponse.deserializeBinary
);


/**
 * @param {!proto.v1.DeleteAgentRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.v1.DeleteAgentResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.v1.DeleteAgentResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServiceClient.prototype.deleteAgent =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/v1.SimulationService/DeleteAgent',
      request,
      metadata,
      methodInfo_SimulationService_DeleteAgent,
      callback);
};


/**
 * @param {!proto.v1.DeleteAgentRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.v1.DeleteAgentResponse>}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServicePromiseClient.prototype.deleteAgent =
    function(request, metadata) {
  return new Promise((resolve, reject) => {
    this.delegateClient_.deleteAgent(
      request, metadata, (error, response) => {
        error ? reject(error) : resolve(response);
      });
  });
};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.v1.ExecuteAgentActionRequest,
 *   !proto.v1.ExecuteAgentActionResponse>}
 */
const methodInfo_SimulationService_ExecuteAgentAction = new grpc.web.AbstractClientBase.MethodInfo(
  proto.v1.ExecuteAgentActionResponse,
  /** @param {!proto.v1.ExecuteAgentActionRequest} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.v1.ExecuteAgentActionResponse.deserializeBinary
);


/**
 * @param {!proto.v1.ExecuteAgentActionRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.v1.ExecuteAgentActionResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.v1.ExecuteAgentActionResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServiceClient.prototype.executeAgentAction =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/v1.SimulationService/ExecuteAgentAction',
      request,
      metadata,
      methodInfo_SimulationService_ExecuteAgentAction,
      callback);
};


/**
 * @param {!proto.v1.ExecuteAgentActionRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.v1.ExecuteAgentActionResponse>}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServicePromiseClient.prototype.executeAgentAction =
    function(request, metadata) {
  return new Promise((resolve, reject) => {
    this.delegateClient_.executeAgentAction(
      request, metadata, (error, response) => {
        error ? reject(error) : resolve(response);
      });
  });
};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.v1.GetAgentObservationRequest,
 *   !proto.v1.GetAgentObservationResponse>}
 */
const methodInfo_SimulationService_GetAgentObservation = new grpc.web.AbstractClientBase.MethodInfo(
  proto.v1.GetAgentObservationResponse,
  /** @param {!proto.v1.GetAgentObservationRequest} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.v1.GetAgentObservationResponse.deserializeBinary
);


/**
 * @param {!proto.v1.GetAgentObservationRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.v1.GetAgentObservationResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.v1.GetAgentObservationResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServiceClient.prototype.getAgentObservation =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/v1.SimulationService/GetAgentObservation',
      request,
      metadata,
      methodInfo_SimulationService_GetAgentObservation,
      callback);
};


/**
 * @param {!proto.v1.GetAgentObservationRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.v1.GetAgentObservationResponse>}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServicePromiseClient.prototype.getAgentObservation =
    function(request, metadata) {
  return new Promise((resolve, reject) => {
    this.delegateClient_.getAgentObservation(
      request, metadata, (error, response) => {
        error ? reject(error) : resolve(response);
      });
  });
};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.v1.CreateSpectatorRequest,
 *   !proto.v1.CellUpdate>}
 */
const methodInfo_SimulationService_CreateSpectator = new grpc.web.AbstractClientBase.MethodInfo(
  proto.v1.CellUpdate,
  /** @param {!proto.v1.CreateSpectatorRequest} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.v1.CellUpdate.deserializeBinary
);


/**
 * @param {!proto.v1.CreateSpectatorRequest} request The request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.v1.CellUpdate>}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServiceClient.prototype.createSpectator =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/v1.SimulationService/CreateSpectator',
      request,
      metadata,
      methodInfo_SimulationService_CreateSpectator);
};


/**
 * @param {!proto.v1.CreateSpectatorRequest} request The request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.v1.CellUpdate>}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServicePromiseClient.prototype.createSpectator =
    function(request, metadata) {
  return this.delegateClient_.client_.serverStreaming(this.delegateClient_.hostname_ +
      '/v1.SimulationService/CreateSpectator',
      request,
      metadata,
      methodInfo_SimulationService_CreateSpectator);
};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.v1.SubscribeSpectatorToRegionRequest,
 *   !proto.v1.SubscribeSpectatorToRegionResponse>}
 */
const methodInfo_SimulationService_SubscribeSpectatorToRegion = new grpc.web.AbstractClientBase.MethodInfo(
  proto.v1.SubscribeSpectatorToRegionResponse,
  /** @param {!proto.v1.SubscribeSpectatorToRegionRequest} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.v1.SubscribeSpectatorToRegionResponse.deserializeBinary
);


/**
 * @param {!proto.v1.SubscribeSpectatorToRegionRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.v1.SubscribeSpectatorToRegionResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.v1.SubscribeSpectatorToRegionResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServiceClient.prototype.subscribeSpectatorToRegion =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/v1.SimulationService/SubscribeSpectatorToRegion',
      request,
      metadata,
      methodInfo_SimulationService_SubscribeSpectatorToRegion,
      callback);
};


/**
 * @param {!proto.v1.SubscribeSpectatorToRegionRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.v1.SubscribeSpectatorToRegionResponse>}
 *     The XHR Node Readable Stream
 */
proto.v1.SimulationServicePromiseClient.prototype.subscribeSpectatorToRegion =
    function(request, metadata) {
  return new Promise((resolve, reject) => {
    this.delegateClient_.subscribeSpectatorToRegion(
      request, metadata, (error, response) => {
        error ? reject(error) : resolve(response);
      });
  });
};


module.exports = proto.v1;

