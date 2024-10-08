// @generated by protoc-gen-es v1.9.0 with parameter "target=ts"
// @generated from file api/v1/common.proto (package api.v1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import type { BinaryReadOptions, FieldList, JsonReadOptions, JsonValue, PartialMessage, PlainMessage } from "@bufbuild/protobuf";
import { Message, proto3 } from "@bufbuild/protobuf";

/**
 * @generated from message api.v1.Void
 */
export class Void extends Message<Void> {
  constructor(data?: PartialMessage<Void>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly runtime: typeof proto3 = proto3;
  static readonly typeName = "api.v1.Void";
  static readonly fields: FieldList = proto3.util.newFieldList(() => [
  ]);

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): Void {
    return new Void().fromBinary(bytes, options);
  }

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): Void {
    return new Void().fromJson(jsonValue, options);
  }

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): Void {
    return new Void().fromJsonString(jsonString, options);
  }

  static equals(a: Void | PlainMessage<Void> | undefined, b: Void | PlainMessage<Void> | undefined): boolean {
    return proto3.util.equals(Void, a, b);
  }
}

