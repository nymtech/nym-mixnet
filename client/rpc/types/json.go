package types


const (
	RequestTypeUnknown = 0
	RequestTypeSend = 1
	RequestTypeFetch = 2
)
//
//type MixConfigJSON struct {
//	Id                   string   `protobuf:"bytes,1,opt,name=Id,json=id,proto3" json:"Id,omitempty"`
//	Host                 string   `protobuf:"bytes,2,opt,name=Host,json=host,proto3" json:"Host,omitempty"`
//	Port                 string   `protobuf:"bytes,3,opt,name=Port,json=port,proto3" json:"Port,omitempty"`
//	PubKey               []byte   `protobuf:"bytes,4,opt,name=PubKey,json=pubKey,proto3" json:"PubKey,omitempty"`
//	Layer                uint64   `protobuf:"varint,5,opt,name=Layer,json=layer,proto3" json:"Layer,omitempty"`
//
//}
//
//type ClientConfigJSON struct {
//	Id                   string     `protobuf:"bytes,1,opt,name=Id,json=id,proto3" json:"Id,omitempty"`
//	Host                 string     `protobuf:"bytes,2,opt,name=Host,json=host,proto3" json:"Host,omitempty"`
//	Port                 string     `protobuf:"bytes,3,opt,name=Port,json=port,proto3" json:"Port,omitempty"`
//	PubKey               []byte     `protobuf:"bytes,4,opt,name=PubKey,json=pubKey,proto3" json:"PubKey,omitempty"`
//	Provider             *MixConfig `protobuf:"bytes,5,opt,name=Provider,json=provider,proto3" json:"Provider,omitempty"`
//
//}
//
//type RequestJson struct {
//	RequestType int `json:"requestType"`
//	SendData string `json:"sendData"` // optional
//	SendDataRecipient ClientConfigJSON `json:"recipient"`// optional
//
//	// we either send: b64 data + recipient
//	// or fetch: <>
//}
//
//type ResponseJson struct {
//	StatusCode int `json:"code"`
//
//	// error or messages or <> (send)
//}