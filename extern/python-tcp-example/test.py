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
    #print("value", value)
    if value < VARINT_TWO_BYTES:
        return value
    elif value <= MAX_UINT16:
        data = await reader.read(2)
    elif value <= MAX_UINT32:
        data = await reader.read(4)
    elif value <= MAX_UINT64:
        data = await reader.read(8)
    return int.from_bytes(data, byteorder="big")

async def send(writer, message):
    data = message.SerializeToString()
    size_data = varint_to_bytes(len(data))
    writer.write(size_data + data)

async def read_response(reader):
    #print("Read one byte...")
    data_size = await read_varint(reader)
    #print(data_size)
    data = await reader.read(data_size)
    #print("Received:", data)

    response = proto.types.Response()
    response.ParseFromString(data)
    return response

async def get_own_details(connection):
    reader, writer = connection

    request_details = proto.types.Request(
        details=proto.types.RequestOwnDetails())
    await send(writer, request_details)

    flush_request = proto.types.Request(
        flush=proto.types.RequestFlush())
    await send(writer, flush_request)

    response = await read_response(reader)
    details = response.details.details

    return details

async def send_message(message, recipient, connection):
    reader, writer = connection

    send_request = proto.types.Request(
        send=proto.types.RequestSendMessage(
            message=message, recipient=recipient))
    await send(writer, send_request)

    flush_request = proto.types.Request(
        flush=proto.types.RequestFlush())
    await send(writer, flush_request)

    flush_response = await read_response(reader)
    print(flush_response)

    send_response = await read_response(reader)
    print(send_response)

async def fetch_messages(connection):
    reader, writer = connection

    fetch_request = proto.types.Request(
        fetch=proto.types.RequestFetchMessages())
    await send(writer, fetch_request)

    flush_request = proto.types.Request(
        flush=proto.types.RequestFlush())
    await send(writer, flush_request)

    flush_response = await read_response(reader)
    print(flush_response)

    fetch_response = await read_response(reader)
    print(fetch_response)

    return fetch_response.fetch.messages

async def run_client():
    connection = await asyncio.open_connection(
        '127.0.0.1', 9001)

    my_details = await get_own_details(connection)
    print(my_details)

    await send_message(b"foomp", my_details, connection)

    messages = await fetch_messages(connection)
    print("Messages:", messages)

    print('Finished')

asyncio.run(run_client())

