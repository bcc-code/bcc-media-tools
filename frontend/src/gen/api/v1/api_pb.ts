// @generated by protoc-gen-es v1.9.0 with parameter "target=ts"
// @generated from file api/v1/api.proto (package api.v1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import type { BinaryReadOptions, FieldList, JsonReadOptions, JsonValue, PartialMessage, PlainMessage } from "@bufbuild/protobuf";
import { Message, proto3 } from "@bufbuild/protobuf";

/**
 * @generated from message api.v1.BMMPermission
 */
export class BMMPermission extends Message<BMMPermission> {
  /**
   * @generated from field: repeated string languages = 1;
   */
  languages: string[] = [];

  /**
   * @generated from field: repeated string albums = 2;
   */
  albums: string[] = [];

  /**
   * @generated from field: repeated string podcasts = 3;
   */
  podcasts: string[] = [];

  /**
   * @generated from field: bool admin = 4;
   */
  admin = false;

  constructor(data?: PartialMessage<BMMPermission>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.BMMPermission";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "languages", kind: "scalar", T: 9 /* ScalarType.STRING */, repeated: true },
    { no: 2, name: "albums", kind: "scalar", T: 9 /* ScalarType.STRING */, repeated: true },
    { no: 3, name: "podcasts", kind: "scalar", T: 9 /* ScalarType.STRING */, repeated: true },
    { no: 4, name: "admin", kind: "scalar", T: 8 /* ScalarType.BOOL */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): BMMPermission {
    return new BMMPermission().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): BMMPermission {
    return new BMMPermission().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): BMMPermission {
    return new BMMPermission().fromJsonString(jsonString, options);
  }

  static equals(a: BMMPermission | PlainMessage<BMMPermission> | undefined, b: BMMPermission | PlainMessage<BMMPermission> | undefined): boolean {
    return proto3.util.equals(BMMPermission, a, b);
  }
}

/**
 * @generated from message api.v1.Permissions
 */
export class Permissions extends Message<Permissions> {
  /**
   * @generated from field: bool admin = 1;
   */
  admin = false;

  /**
   * @generated from field: api.v1.BMMPermission bmm = 2;
   */
  bmm?: BMMPermission;

  constructor(data?: PartialMessage<Permissions>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.Permissions";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "admin", kind: "scalar", T: 8 /* ScalarType.BOOL */ },
    { no: 2, name: "bmm", kind: "message", T: BMMPermission },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): Permissions {
    return new Permissions().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): Permissions {
    return new Permissions().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): Permissions {
    return new Permissions().fromJsonString(jsonString, options);
  }

  static equals(a: Permissions | PlainMessage<Permissions> | undefined, b: Permissions | PlainMessage<Permissions> | undefined): boolean {
    return proto3.util.equals(Permissions, a, b);
  }
}

/**
 * @generated from message api.v1.GetPermissionsRequest
 */
export class GetPermissionsRequest extends Message<GetPermissionsRequest> {
  constructor(data?: PartialMessage<GetPermissionsRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.GetPermissionsRequest";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): GetPermissionsRequest {
    return new GetPermissionsRequest().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): GetPermissionsRequest {
    return new GetPermissionsRequest().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): GetPermissionsRequest {
    return new GetPermissionsRequest().fromJsonString(jsonString, options);
  }

  static equals(a: GetPermissionsRequest | PlainMessage<GetPermissionsRequest> | undefined, b: GetPermissionsRequest | PlainMessage<GetPermissionsRequest> | undefined): boolean {
    return proto3.util.equals(GetPermissionsRequest, a, b);
  }
}

/**
 * @generated from message api.v1.SetPermissionsRequest
 */
export class SetPermissionsRequest extends Message<SetPermissionsRequest> {
  /**
   * @generated from field: string email = 1;
   */
  email = "";

  /**
   * @generated from field: api.v1.Permissions permissions = 2;
   */
  permissions?: Permissions;

  constructor(data?: PartialMessage<SetPermissionsRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.SetPermissionsRequest";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "email", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "permissions", kind: "message", T: Permissions },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): SetPermissionsRequest {
    return new SetPermissionsRequest().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): SetPermissionsRequest {
    return new SetPermissionsRequest().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): SetPermissionsRequest {
    return new SetPermissionsRequest().fromJsonString(jsonString, options);
  }

  static equals(a: SetPermissionsRequest | PlainMessage<SetPermissionsRequest> | undefined, b: SetPermissionsRequest | PlainMessage<SetPermissionsRequest> | undefined): boolean {
    return proto3.util.equals(SetPermissionsRequest, a, b);
  }
}

/**
 * @generated from message api.v1.DeletePermissionsRequest
 */
export class DeletePermissionsRequest extends Message<DeletePermissionsRequest> {
  /**
   * @generated from field: string email = 1;
   */
  email = "";

  constructor(data?: PartialMessage<DeletePermissionsRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.DeletePermissionsRequest";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "email", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): DeletePermissionsRequest {
    return new DeletePermissionsRequest().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): DeletePermissionsRequest {
    return new DeletePermissionsRequest().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): DeletePermissionsRequest {
    return new DeletePermissionsRequest().fromJsonString(jsonString, options);
  }

  static equals(a: DeletePermissionsRequest | PlainMessage<DeletePermissionsRequest> | undefined, b: DeletePermissionsRequest | PlainMessage<DeletePermissionsRequest> | undefined): boolean {
    return proto3.util.equals(DeletePermissionsRequest, a, b);
  }
}

/**
 * @generated from message api.v1.PermissionsList
 */
export class PermissionsList extends Message<PermissionsList> {
  /**
   * @generated from field: map<string, api.v1.Permissions> permissions = 1;
   */
  permissions: { [key: string]: Permissions } = {};

  constructor(data?: PartialMessage<PermissionsList>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.PermissionsList";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "permissions", kind: "map", K: 9 /* ScalarType.STRING */, V: {kind: "message", T: Permissions} },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): PermissionsList {
    return new PermissionsList().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): PermissionsList {
    return new PermissionsList().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): PermissionsList {
    return new PermissionsList().fromJsonString(jsonString, options);
  }

  static equals(a: PermissionsList | PlainMessage<PermissionsList> | undefined, b: PermissionsList | PlainMessage<PermissionsList> | undefined): boolean {
    return proto3.util.equals(PermissionsList, a, b);
  }
}

/**
 * @generated from message api.v1.BMMYear
 */
export class BMMYear extends Message<BMMYear> {
  /**
   * @generated from field: uint32 year = 1;
   */
  year = 0;

  /**
   * @generated from field: uint32 count = 2;
   */
  count = 0;

  constructor(data?: PartialMessage<BMMYear>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.BMMYear";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "year", kind: "scalar", T: 13 /* ScalarType.UINT32 */ },
    { no: 2, name: "count", kind: "scalar", T: 13 /* ScalarType.UINT32 */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): BMMYear {
    return new BMMYear().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): BMMYear {
    return new BMMYear().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): BMMYear {
    return new BMMYear().fromJsonString(jsonString, options);
  }

  static equals(a: BMMYear | PlainMessage<BMMYear> | undefined, b: BMMYear | PlainMessage<BMMYear> | undefined): boolean {
    return proto3.util.equals(BMMYear, a, b);
  }
}

/**
 * @generated from message api.v1.GetYearsResponse
 */
export class GetYearsResponse extends Message<GetYearsResponse> {
  /**
   * @generated from field: map<uint32, api.v1.BMMYear> data = 1;
   */
  data: { [key: number]: BMMYear } = {};

  constructor(data?: PartialMessage<GetYearsResponse>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.GetYearsResponse";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "data", kind: "map", K: 13 /* ScalarType.UINT32 */, V: {kind: "message", T: BMMYear} },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): GetYearsResponse {
    return new GetYearsResponse().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): GetYearsResponse {
    return new GetYearsResponse().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): GetYearsResponse {
    return new GetYearsResponse().fromJsonString(jsonString, options);
  }

  static equals(a: GetYearsResponse | PlainMessage<GetYearsResponse> | undefined, b: GetYearsResponse | PlainMessage<GetYearsResponse> | undefined): boolean {
    return proto3.util.equals(GetYearsResponse, a, b);
  }
}

/**
 * @generated from message api.v1.GetAlbumsRequest
 */
export class GetAlbumsRequest extends Message<GetAlbumsRequest> {
  /**
   * @generated from field: uint32 year = 1;
   */
  year = 0;

  constructor(data?: PartialMessage<GetAlbumsRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.GetAlbumsRequest";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "year", kind: "scalar", T: 13 /* ScalarType.UINT32 */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): GetAlbumsRequest {
    return new GetAlbumsRequest().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): GetAlbumsRequest {
    return new GetAlbumsRequest().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): GetAlbumsRequest {
    return new GetAlbumsRequest().fromJsonString(jsonString, options);
  }

  static equals(a: GetAlbumsRequest | PlainMessage<GetAlbumsRequest> | undefined, b: GetAlbumsRequest | PlainMessage<GetAlbumsRequest> | undefined): boolean {
    return proto3.util.equals(GetAlbumsRequest, a, b);
  }
}

/**
 * @generated from message api.v1.Album
 */
export class Album extends Message<Album> {
  /**
   * @generated from field: string id = 1;
   */
  id = "";

  /**
   * @generated from field: string title = 2;
   */
  title = "";

  /**
   * @generated from field: string cover = 4;
   */
  cover = "";

  /**
   * @generated from field: repeated string languages = 5;
   */
  languages: string[] = [];

  constructor(data?: PartialMessage<Album>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.Album";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "title", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 4, name: "cover", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 5, name: "languages", kind: "scalar", T: 9 /* ScalarType.STRING */, repeated: true },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): Album {
    return new Album().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): Album {
    return new Album().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): Album {
    return new Album().fromJsonString(jsonString, options);
  }

  static equals(a: Album | PlainMessage<Album> | undefined, b: Album | PlainMessage<Album> | undefined): boolean {
    return proto3.util.equals(Album, a, b);
  }
}

/**
 * @generated from message api.v1.AlbumsList
 */
export class AlbumsList extends Message<AlbumsList> {
  /**
   * @generated from field: repeated api.v1.Album albums = 1;
   */
  albums: Album[] = [];

  constructor(data?: PartialMessage<AlbumsList>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.AlbumsList";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "albums", kind: "message", T: Album, repeated: true },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): AlbumsList {
    return new AlbumsList().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): AlbumsList {
    return new AlbumsList().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): AlbumsList {
    return new AlbumsList().fromJsonString(jsonString, options);
  }

  static equals(a: AlbumsList | PlainMessage<AlbumsList> | undefined, b: AlbumsList | PlainMessage<AlbumsList> | undefined): boolean {
    return proto3.util.equals(AlbumsList, a, b);
  }
}

/**
 * @generated from message api.v1.GetAlbumTracksRequest
 */
export class GetAlbumTracksRequest extends Message<GetAlbumTracksRequest> {
  /**
   * @generated from field: string album_id = 1;
   */
  albumId = "";

  constructor(data?: PartialMessage<GetAlbumTracksRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.GetAlbumTracksRequest";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "album_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): GetAlbumTracksRequest {
    return new GetAlbumTracksRequest().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): GetAlbumTracksRequest {
    return new GetAlbumTracksRequest().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): GetAlbumTracksRequest {
    return new GetAlbumTracksRequest().fromJsonString(jsonString, options);
  }

  static equals(a: GetAlbumTracksRequest | PlainMessage<GetAlbumTracksRequest> | undefined, b: GetAlbumTracksRequest | PlainMessage<GetAlbumTracksRequest> | undefined): boolean {
    return proto3.util.equals(GetAlbumTracksRequest, a, b);
  }
}

/**
 * @generated from message api.v1.GetPodcastTracksRequest
 */
export class GetPodcastTracksRequest extends Message<GetPodcastTracksRequest> {
  /**
   * @generated from field: string podcast_tag = 1;
   */
  podcastTag = "";

  /**
   * @generated from field: uint32 limit = 2;
   */
  limit = 0;

  constructor(data?: PartialMessage<GetPodcastTracksRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.GetPodcastTracksRequest";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "podcast_tag", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "limit", kind: "scalar", T: 13 /* ScalarType.UINT32 */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): GetPodcastTracksRequest {
    return new GetPodcastTracksRequest().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): GetPodcastTracksRequest {
    return new GetPodcastTracksRequest().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): GetPodcastTracksRequest {
    return new GetPodcastTracksRequest().fromJsonString(jsonString, options);
  }

  static equals(a: GetPodcastTracksRequest | PlainMessage<GetPodcastTracksRequest> | undefined, b: GetPodcastTracksRequest | PlainMessage<GetPodcastTracksRequest> | undefined): boolean {
    return proto3.util.equals(GetPodcastTracksRequest, a, b);
  }
}

/**
 * @generated from message api.v1.BMMTrack
 */
export class BMMTrack extends Message<BMMTrack> {
  /**
   * @generated from field: string id = 1;
   */
  id = "";

  /**
   * @generated from field: string title = 2;
   */
  title = "";

  constructor(data?: PartialMessage<BMMTrack>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.BMMTrack";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "title", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): BMMTrack {
    return new BMMTrack().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): BMMTrack {
    return new BMMTrack().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): BMMTrack {
    return new BMMTrack().fromJsonString(jsonString, options);
  }

  static equals(a: BMMTrack | PlainMessage<BMMTrack> | undefined, b: BMMTrack | PlainMessage<BMMTrack> | undefined): boolean {
    return proto3.util.equals(BMMTrack, a, b);
  }
}

/**
 * @generated from message api.v1.TracksList
 */
export class TracksList extends Message<TracksList> {
  /**
   * @generated from field: repeated api.v1.BMMTrack tracks = 1;
   */
  tracks: BMMTrack[] = [];

  constructor(data?: PartialMessage<TracksList>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.TracksList";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "tracks", kind: "message", T: BMMTrack, repeated: true },
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): TracksList {
    return new TracksList().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): TracksList {
    return new TracksList().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): TracksList {
    return new TracksList().fromJsonString(jsonString, options);
  }

  static equals(a: TracksList | PlainMessage<TracksList> | undefined, b: TracksList | PlainMessage<TracksList> | undefined): boolean {
    return proto3.util.equals(TracksList, a, b);
  }
}
