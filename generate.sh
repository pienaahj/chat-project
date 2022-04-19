protoc  --go_out=. -I=server/chatpb server/chatpb/chat.proto \
        --go-grpc_out=. -I=server/chatpb server/chatpb/chat.proto \
        