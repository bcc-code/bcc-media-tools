// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.0
// 	protoc        (unknown)
// source: api/v1/api.proto

package apiv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type BMMPermission struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Languages []string `protobuf:"bytes,1,rep,name=languages,proto3" json:"languages,omitempty"`
	Albums    []string `protobuf:"bytes,2,rep,name=albums,proto3" json:"albums,omitempty"`
}

func (x *BMMPermission) Reset() {
	*x = BMMPermission{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BMMPermission) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BMMPermission) ProtoMessage() {}

func (x *BMMPermission) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BMMPermission.ProtoReflect.Descriptor instead.
func (*BMMPermission) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{0}
}

func (x *BMMPermission) GetLanguages() []string {
	if x != nil {
		return x.Languages
	}
	return nil
}

func (x *BMMPermission) GetAlbums() []string {
	if x != nil {
		return x.Albums
	}
	return nil
}

type Permissions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Admin bool           `protobuf:"varint,1,opt,name=admin,proto3" json:"admin,omitempty"`
	Bmm   *BMMPermission `protobuf:"bytes,2,opt,name=bmm,proto3" json:"bmm,omitempty"`
}

func (x *Permissions) Reset() {
	*x = Permissions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Permissions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Permissions) ProtoMessage() {}

func (x *Permissions) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Permissions.ProtoReflect.Descriptor instead.
func (*Permissions) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{1}
}

func (x *Permissions) GetAdmin() bool {
	if x != nil {
		return x.Admin
	}
	return false
}

func (x *Permissions) GetBmm() *BMMPermission {
	if x != nil {
		return x.Bmm
	}
	return nil
}

type GetPermissionsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetPermissionsRequest) Reset() {
	*x = GetPermissionsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetPermissionsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPermissionsRequest) ProtoMessage() {}

func (x *GetPermissionsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPermissionsRequest.ProtoReflect.Descriptor instead.
func (*GetPermissionsRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{2}
}

type SetPermissionsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Email       string       `protobuf:"bytes,1,opt,name=email,proto3" json:"email,omitempty"`
	Permissions *Permissions `protobuf:"bytes,2,opt,name=permissions,proto3" json:"permissions,omitempty"`
}

func (x *SetPermissionsRequest) Reset() {
	*x = SetPermissionsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetPermissionsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetPermissionsRequest) ProtoMessage() {}

func (x *SetPermissionsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetPermissionsRequest.ProtoReflect.Descriptor instead.
func (*SetPermissionsRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{3}
}

func (x *SetPermissionsRequest) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *SetPermissionsRequest) GetPermissions() *Permissions {
	if x != nil {
		return x.Permissions
	}
	return nil
}

type DeletePermissionsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Email string `protobuf:"bytes,1,opt,name=email,proto3" json:"email,omitempty"`
}

func (x *DeletePermissionsRequest) Reset() {
	*x = DeletePermissionsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeletePermissionsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeletePermissionsRequest) ProtoMessage() {}

func (x *DeletePermissionsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeletePermissionsRequest.ProtoReflect.Descriptor instead.
func (*DeletePermissionsRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{4}
}

func (x *DeletePermissionsRequest) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

type PermissionsList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Permissions map[string]*Permissions `protobuf:"bytes,1,rep,name=permissions,proto3" json:"permissions,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *PermissionsList) Reset() {
	*x = PermissionsList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PermissionsList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PermissionsList) ProtoMessage() {}

func (x *PermissionsList) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PermissionsList.ProtoReflect.Descriptor instead.
func (*PermissionsList) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{5}
}

func (x *PermissionsList) GetPermissions() map[string]*Permissions {
	if x != nil {
		return x.Permissions
	}
	return nil
}

type BMMYear struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Year  uint32 `protobuf:"varint,1,opt,name=year,proto3" json:"year,omitempty"`
	Count uint32 `protobuf:"varint,2,opt,name=count,proto3" json:"count,omitempty"`
}

func (x *BMMYear) Reset() {
	*x = BMMYear{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BMMYear) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BMMYear) ProtoMessage() {}

func (x *BMMYear) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BMMYear.ProtoReflect.Descriptor instead.
func (*BMMYear) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{6}
}

func (x *BMMYear) GetYear() uint32 {
	if x != nil {
		return x.Year
	}
	return 0
}

func (x *BMMYear) GetCount() uint32 {
	if x != nil {
		return x.Count
	}
	return 0
}

type GetYearsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data map[uint32]*BMMYear `protobuf:"bytes,1,rep,name=data,proto3" json:"data,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *GetYearsResponse) Reset() {
	*x = GetYearsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetYearsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetYearsResponse) ProtoMessage() {}

func (x *GetYearsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetYearsResponse.ProtoReflect.Descriptor instead.
func (*GetYearsResponse) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{7}
}

func (x *GetYearsResponse) GetData() map[uint32]*BMMYear {
	if x != nil {
		return x.Data
	}
	return nil
}

type GetAlbumsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Year uint32 `protobuf:"varint,1,opt,name=year,proto3" json:"year,omitempty"`
}

func (x *GetAlbumsRequest) Reset() {
	*x = GetAlbumsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAlbumsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAlbumsRequest) ProtoMessage() {}

func (x *GetAlbumsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAlbumsRequest.ProtoReflect.Descriptor instead.
func (*GetAlbumsRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{8}
}

func (x *GetAlbumsRequest) GetYear() uint32 {
	if x != nil {
		return x.Year
	}
	return 0
}

type Album struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id        string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Title     string   `protobuf:"bytes,2,opt,name=title,proto3" json:"title,omitempty"`
	Cover     string   `protobuf:"bytes,4,opt,name=cover,proto3" json:"cover,omitempty"`
	Languages []string `protobuf:"bytes,5,rep,name=languages,proto3" json:"languages,omitempty"`
}

func (x *Album) Reset() {
	*x = Album{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Album) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Album) ProtoMessage() {}

func (x *Album) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Album.ProtoReflect.Descriptor instead.
func (*Album) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{9}
}

func (x *Album) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Album) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

func (x *Album) GetCover() string {
	if x != nil {
		return x.Cover
	}
	return ""
}

func (x *Album) GetLanguages() []string {
	if x != nil {
		return x.Languages
	}
	return nil
}

type AlbumsList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Albums []*Album `protobuf:"bytes,1,rep,name=albums,proto3" json:"albums,omitempty"`
}

func (x *AlbumsList) Reset() {
	*x = AlbumsList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AlbumsList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AlbumsList) ProtoMessage() {}

func (x *AlbumsList) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AlbumsList.ProtoReflect.Descriptor instead.
func (*AlbumsList) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{10}
}

func (x *AlbumsList) GetAlbums() []*Album {
	if x != nil {
		return x.Albums
	}
	return nil
}

type GetAlbumTracksRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AlbumId string `protobuf:"bytes,1,opt,name=album_id,json=albumId,proto3" json:"album_id,omitempty"`
}

func (x *GetAlbumTracksRequest) Reset() {
	*x = GetAlbumTracksRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[11]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAlbumTracksRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAlbumTracksRequest) ProtoMessage() {}

func (x *GetAlbumTracksRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[11]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAlbumTracksRequest.ProtoReflect.Descriptor instead.
func (*GetAlbumTracksRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{11}
}

func (x *GetAlbumTracksRequest) GetAlbumId() string {
	if x != nil {
		return x.AlbumId
	}
	return ""
}

type GetPodcastTracksRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PodcastTag string `protobuf:"bytes,1,opt,name=podcast_tag,json=podcastTag,proto3" json:"podcast_tag,omitempty"`
	Limit      uint32 `protobuf:"varint,2,opt,name=limit,proto3" json:"limit,omitempty"`
}

func (x *GetPodcastTracksRequest) Reset() {
	*x = GetPodcastTracksRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[12]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetPodcastTracksRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPodcastTracksRequest) ProtoMessage() {}

func (x *GetPodcastTracksRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[12]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPodcastTracksRequest.ProtoReflect.Descriptor instead.
func (*GetPodcastTracksRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{12}
}

func (x *GetPodcastTracksRequest) GetPodcastTag() string {
	if x != nil {
		return x.PodcastTag
	}
	return ""
}

func (x *GetPodcastTracksRequest) GetLimit() uint32 {
	if x != nil {
		return x.Limit
	}
	return 0
}

type BMMTrack struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id    string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Title string `protobuf:"bytes,2,opt,name=title,proto3" json:"title,omitempty"`
}

func (x *BMMTrack) Reset() {
	*x = BMMTrack{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[13]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BMMTrack) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BMMTrack) ProtoMessage() {}

func (x *BMMTrack) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[13]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BMMTrack.ProtoReflect.Descriptor instead.
func (*BMMTrack) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{13}
}

func (x *BMMTrack) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *BMMTrack) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

type TracksList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tracks []*BMMTrack `protobuf:"bytes,1,rep,name=tracks,proto3" json:"tracks,omitempty"`
}

func (x *TracksList) Reset() {
	*x = TracksList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_v1_api_proto_msgTypes[14]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TracksList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TracksList) ProtoMessage() {}

func (x *TracksList) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_api_proto_msgTypes[14]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TracksList.ProtoReflect.Descriptor instead.
func (*TracksList) Descriptor() ([]byte, []int) {
	return file_api_v1_api_proto_rawDescGZIP(), []int{14}
}

func (x *TracksList) GetTracks() []*BMMTrack {
	if x != nil {
		return x.Tracks
	}
	return nil
}

var File_api_v1_api_proto protoreflect.FileDescriptor

var file_api_v1_api_proto_rawDesc = []byte{
	0x0a, 0x10, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x06, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x1a, 0x13, 0x61, 0x70, 0x69, 0x2f,
	0x76, 0x31, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x45, 0x0a, 0x0d, 0x42, 0x4d, 0x4d, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e,
	0x12, 0x1c, 0x0a, 0x09, 0x6c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x09, 0x6c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x73, 0x12, 0x16,
	0x0a, 0x06, 0x61, 0x6c, 0x62, 0x75, 0x6d, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06,
	0x61, 0x6c, 0x62, 0x75, 0x6d, 0x73, 0x22, 0x4c, 0x0a, 0x0b, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x12, 0x27, 0x0a, 0x03, 0x62,
	0x6d, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76,
	0x31, 0x2e, 0x42, 0x4d, 0x4d, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x52,
	0x03, 0x62, 0x6d, 0x6d, 0x22, 0x17, 0x0a, 0x15, 0x47, 0x65, 0x74, 0x50, 0x65, 0x72, 0x6d, 0x69,
	0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x64, 0x0a,
	0x15, 0x53, 0x65, 0x74, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x35, 0x0a, 0x0b,
	0x70, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x13, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x65, 0x72, 0x6d, 0x69,
	0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x0b, 0x70, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x73, 0x22, 0x30, 0x0a, 0x18, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x50, 0x65, 0x72,
	0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x14, 0x0a, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x65, 0x6d, 0x61, 0x69, 0x6c, 0x22, 0xb2, 0x01, 0x0a, 0x0f, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x4a, 0x0a, 0x0b, 0x70, 0x65, 0x72,
	0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x28,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x2e, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0b, 0x70, 0x65, 0x72, 0x6d, 0x69, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x53, 0x0a, 0x10, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73,
	0x69, 0x6f, 0x6e, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x29, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x76, 0x31, 0x2e, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x33, 0x0a, 0x07, 0x42, 0x4d,
	0x4d, 0x59, 0x65, 0x61, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x79, 0x65, 0x61, 0x72, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x04, 0x79, 0x65, 0x61, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x22,
	0x94, 0x01, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x59, 0x65, 0x61, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x36, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x22, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x59,
	0x65, 0x61, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x44, 0x61, 0x74,
	0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x1a, 0x48, 0x0a, 0x09,
	0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x25, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x76, 0x31, 0x2e, 0x42, 0x4d, 0x4d, 0x59, 0x65, 0x61, 0x72, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x26, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x41, 0x6c, 0x62,
	0x75, 0x6d, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x79, 0x65,
	0x61, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x79, 0x65, 0x61, 0x72, 0x22, 0x61,
	0x0a, 0x05, 0x41, 0x6c, 0x62, 0x75, 0x6d, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x69, 0x74, 0x6c, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x12, 0x14, 0x0a,
	0x05, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x63, 0x6f,
	0x76, 0x65, 0x72, 0x12, 0x1c, 0x0a, 0x09, 0x6c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x73,
	0x18, 0x05, 0x20, 0x03, 0x28, 0x09, 0x52, 0x09, 0x6c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65,
	0x73, 0x22, 0x33, 0x0a, 0x0a, 0x41, 0x6c, 0x62, 0x75, 0x6d, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x12,
	0x25, 0x0a, 0x06, 0x61, 0x6c, 0x62, 0x75, 0x6d, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x0d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x6c, 0x62, 0x75, 0x6d, 0x52, 0x06,
	0x61, 0x6c, 0x62, 0x75, 0x6d, 0x73, 0x22, 0x32, 0x0a, 0x15, 0x47, 0x65, 0x74, 0x41, 0x6c, 0x62,
	0x75, 0x6d, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x19, 0x0a, 0x08, 0x61, 0x6c, 0x62, 0x75, 0x6d, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x61, 0x6c, 0x62, 0x75, 0x6d, 0x49, 0x64, 0x22, 0x50, 0x0a, 0x17, 0x47, 0x65,
	0x74, 0x50, 0x6f, 0x64, 0x63, 0x61, 0x73, 0x74, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x73, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x70, 0x6f, 0x64, 0x63, 0x61, 0x73, 0x74,
	0x5f, 0x74, 0x61, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x70, 0x6f, 0x64, 0x63,
	0x61, 0x73, 0x74, 0x54, 0x61, 0x67, 0x12, 0x14, 0x0a, 0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x22, 0x30, 0x0a, 0x08,
	0x42, 0x4d, 0x4d, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x69, 0x74, 0x6c,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x22, 0x36,
	0x0a, 0x0a, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x28, 0x0a, 0x06,
	0x74, 0x72, 0x61, 0x63, 0x6b, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x4d, 0x4d, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x52, 0x06,
	0x74, 0x72, 0x61, 0x63, 0x6b, 0x73, 0x32, 0x8f, 0x04, 0x0a, 0x0a, 0x41, 0x50, 0x49, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x35, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x50, 0x65, 0x72, 0x6d,
	0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x0c, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31,
	0x2e, 0x56, 0x6f, 0x69, 0x64, 0x1a, 0x13, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x50,
	0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0x00, 0x12, 0x42, 0x0a, 0x11,
	0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e,
	0x73, 0x12, 0x1d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x74, 0x50, 0x65,
	0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x0c, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x56, 0x6f, 0x69, 0x64, 0x22, 0x00,
	0x12, 0x45, 0x0a, 0x11, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73,
	0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x20, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x44,
	0x65, 0x6c, 0x65, 0x74, 0x65, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0c, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31,
	0x2e, 0x56, 0x6f, 0x69, 0x64, 0x22, 0x00, 0x12, 0x3a, 0x0a, 0x0f, 0x4c, 0x69, 0x73, 0x74, 0x50,
	0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x0c, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x76, 0x31, 0x2e, 0x56, 0x6f, 0x69, 0x64, 0x1a, 0x17, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76,
	0x31, 0x2e, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x4c, 0x69, 0x73,
	0x74, 0x22, 0x00, 0x12, 0x34, 0x0a, 0x08, 0x47, 0x65, 0x74, 0x59, 0x65, 0x61, 0x72, 0x73, 0x12,
	0x0c, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x56, 0x6f, 0x69, 0x64, 0x1a, 0x18, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x59, 0x65, 0x61, 0x72, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x3b, 0x0a, 0x09, 0x47, 0x65, 0x74,
	0x41, 0x6c, 0x62, 0x75, 0x6d, 0x73, 0x12, 0x18, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e,
	0x47, 0x65, 0x74, 0x41, 0x6c, 0x62, 0x75, 0x6d, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x6c, 0x62, 0x75, 0x6d, 0x73,
	0x4c, 0x69, 0x73, 0x74, 0x22, 0x00, 0x12, 0x45, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x41, 0x6c, 0x62,
	0x75, 0x6d, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x73, 0x12, 0x1d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76,
	0x31, 0x2e, 0x47, 0x65, 0x74, 0x41, 0x6c, 0x62, 0x75, 0x6d, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x73,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31,
	0x2e, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x22, 0x00, 0x12, 0x49, 0x0a,
	0x10, 0x47, 0x65, 0x74, 0x50, 0x6f, 0x64, 0x63, 0x61, 0x73, 0x74, 0x54, 0x72, 0x61, 0x63, 0x6b,
	0x73, 0x12, 0x1f, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x50, 0x6f,
	0x64, 0x63, 0x61, 0x73, 0x74, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x72, 0x61, 0x63,
	0x6b, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x22, 0x00, 0x42, 0x22, 0x5a, 0x20, 0x62, 0x63, 0x63, 0x2d,
	0x6d, 0x65, 0x64, 0x69, 0x61, 0x2d, 0x74, 0x6f, 0x6f, 0x6c, 0x73, 0x2f, 0x67, 0x65, 0x6e, 0x2f,
	0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x3b, 0x61, 0x70, 0x69, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_v1_api_proto_rawDescOnce sync.Once
	file_api_v1_api_proto_rawDescData = file_api_v1_api_proto_rawDesc
)

func file_api_v1_api_proto_rawDescGZIP() []byte {
	file_api_v1_api_proto_rawDescOnce.Do(func() {
		file_api_v1_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_v1_api_proto_rawDescData)
	})
	return file_api_v1_api_proto_rawDescData
}

var file_api_v1_api_proto_msgTypes = make([]protoimpl.MessageInfo, 17)
var file_api_v1_api_proto_goTypes = []interface{}{
	(*BMMPermission)(nil),            // 0: api.v1.BMMPermission
	(*Permissions)(nil),              // 1: api.v1.Permissions
	(*GetPermissionsRequest)(nil),    // 2: api.v1.GetPermissionsRequest
	(*SetPermissionsRequest)(nil),    // 3: api.v1.SetPermissionsRequest
	(*DeletePermissionsRequest)(nil), // 4: api.v1.DeletePermissionsRequest
	(*PermissionsList)(nil),          // 5: api.v1.PermissionsList
	(*BMMYear)(nil),                  // 6: api.v1.BMMYear
	(*GetYearsResponse)(nil),         // 7: api.v1.GetYearsResponse
	(*GetAlbumsRequest)(nil),         // 8: api.v1.GetAlbumsRequest
	(*Album)(nil),                    // 9: api.v1.Album
	(*AlbumsList)(nil),               // 10: api.v1.AlbumsList
	(*GetAlbumTracksRequest)(nil),    // 11: api.v1.GetAlbumTracksRequest
	(*GetPodcastTracksRequest)(nil),  // 12: api.v1.GetPodcastTracksRequest
	(*BMMTrack)(nil),                 // 13: api.v1.BMMTrack
	(*TracksList)(nil),               // 14: api.v1.TracksList
	nil,                              // 15: api.v1.PermissionsList.PermissionsEntry
	nil,                              // 16: api.v1.GetYearsResponse.DataEntry
	(*Void)(nil),                     // 17: api.v1.Void
}
var file_api_v1_api_proto_depIdxs = []int32{
	0,  // 0: api.v1.Permissions.bmm:type_name -> api.v1.BMMPermission
	1,  // 1: api.v1.SetPermissionsRequest.permissions:type_name -> api.v1.Permissions
	15, // 2: api.v1.PermissionsList.permissions:type_name -> api.v1.PermissionsList.PermissionsEntry
	16, // 3: api.v1.GetYearsResponse.data:type_name -> api.v1.GetYearsResponse.DataEntry
	9,  // 4: api.v1.AlbumsList.albums:type_name -> api.v1.Album
	13, // 5: api.v1.TracksList.tracks:type_name -> api.v1.BMMTrack
	1,  // 6: api.v1.PermissionsList.PermissionsEntry.value:type_name -> api.v1.Permissions
	6,  // 7: api.v1.GetYearsResponse.DataEntry.value:type_name -> api.v1.BMMYear
	17, // 8: api.v1.APIService.GetPermissions:input_type -> api.v1.Void
	3,  // 9: api.v1.APIService.UpdatePermissions:input_type -> api.v1.SetPermissionsRequest
	4,  // 10: api.v1.APIService.DeletePermissions:input_type -> api.v1.DeletePermissionsRequest
	17, // 11: api.v1.APIService.ListPermissions:input_type -> api.v1.Void
	17, // 12: api.v1.APIService.GetYears:input_type -> api.v1.Void
	8,  // 13: api.v1.APIService.GetAlbums:input_type -> api.v1.GetAlbumsRequest
	11, // 14: api.v1.APIService.GetAlbumTracks:input_type -> api.v1.GetAlbumTracksRequest
	12, // 15: api.v1.APIService.GetPodcastTracks:input_type -> api.v1.GetPodcastTracksRequest
	1,  // 16: api.v1.APIService.GetPermissions:output_type -> api.v1.Permissions
	17, // 17: api.v1.APIService.UpdatePermissions:output_type -> api.v1.Void
	17, // 18: api.v1.APIService.DeletePermissions:output_type -> api.v1.Void
	5,  // 19: api.v1.APIService.ListPermissions:output_type -> api.v1.PermissionsList
	7,  // 20: api.v1.APIService.GetYears:output_type -> api.v1.GetYearsResponse
	10, // 21: api.v1.APIService.GetAlbums:output_type -> api.v1.AlbumsList
	14, // 22: api.v1.APIService.GetAlbumTracks:output_type -> api.v1.TracksList
	14, // 23: api.v1.APIService.GetPodcastTracks:output_type -> api.v1.TracksList
	16, // [16:24] is the sub-list for method output_type
	8,  // [8:16] is the sub-list for method input_type
	8,  // [8:8] is the sub-list for extension type_name
	8,  // [8:8] is the sub-list for extension extendee
	0,  // [0:8] is the sub-list for field type_name
}

func init() { file_api_v1_api_proto_init() }
func file_api_v1_api_proto_init() {
	if File_api_v1_api_proto != nil {
		return
	}
	file_api_v1_common_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_api_v1_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BMMPermission); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Permissions); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetPermissionsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SetPermissionsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeletePermissionsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PermissionsList); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BMMYear); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetYearsResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAlbumsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Album); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AlbumsList); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[11].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAlbumTracksRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[12].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetPodcastTracksRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[13].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BMMTrack); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_v1_api_proto_msgTypes[14].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TracksList); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_v1_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   17,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_v1_api_proto_goTypes,
		DependencyIndexes: file_api_v1_api_proto_depIdxs,
		MessageInfos:      file_api_v1_api_proto_msgTypes,
	}.Build()
	File_api_v1_api_proto = out.File
	file_api_v1_api_proto_rawDesc = nil
	file_api_v1_api_proto_goTypes = nil
	file_api_v1_api_proto_depIdxs = nil
}
