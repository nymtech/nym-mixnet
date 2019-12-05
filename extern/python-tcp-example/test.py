import asyncio
import proto
import struct

VARINT_TWO_BYTES = 0xfd
VARINT_FOUR_BYTES = 0xfe
VARINT_EIGHT_BYTES = 0xff
MAX_UINT16 = 2**16 - 1
MAX_UINT32 = 2**32 - 1
MAX_UINT64 = 2**64 - 1
assert MAX_UINT16 == 0xffff
assert MAX_UINT32 == 0xffffffff
assert MAX_UINT64 == 0xffffffffffffffff

def varint_to_bytes(value):
    if value < VARINT_TWO_BYTES:
        return bytes([value])
    elif value <= MAX_UINT16:
        return (bytes([VARINT_TWO_BYTES]) +
                value.to_bytes(2, byteorder="big"))
    elif value <= MAX_UINT32:
        return (bytes([VARINT_FOUR_BYTES]) +
                value.to_bytes(4, byteorder="big"))
    elif value <= MAX_UINT64:
        return (bytes([VARINT_EIGHT_BYTES]) +
                value.to_bytes(8, byteorder="big"))

async def read_varint(reader):
    value = await reader.read(1)
    value = int.from_bytes(value, byteorder="big")
    if value < VARINT_TWO_BYTES:
        return value
    elif value <= MAX_UINT16:
        data = await reader.read(2)
    elif value <= MAX_UINT32:
        data = await reader.read(4)
    elif value <= MAX_UINT64:
        data = await reader.read(8)
    return int.from_bytes(data, byteorder="big")

async def run_client():
    reader, writer = await asyncio.open_connection(
        '127.0.0.1', 9001)

    request_details = proto.types.Request(
        details=proto.types.RequestOwnDetails())
    data = request_details.SerializeToString()
    size_data = varint_to_bytes(len(data))
    writer.write(size_data + data)

    flush_request = proto.types.Request(
        flush=proto.types.RequestFlush())
    data = flush_request.SerializeToString()
    size_data = varint_to_bytes(len(data))
    writer.write(size_data + data)

    #print("Read one byte...")
    data_size = await read_varint(reader)
    #print(data_size)
    data = await reader.read(data_size)
    #print("Received:", data)

    response = proto.types.Response()
    response.ParseFromString(data)
    print(response)

    print('Close the connection')
    writer.close()

asyncio.run(run_client())

