import asyncio
import proto
import util

async def send(writer, message):
    data = message.SerializeToString()
    size_data = util.varint_to_bytes(len(data))
    writer.write(size_data + data)

async def read_response(reader):
    data_size = await util.read_varint(reader)
    data = await reader.read(data_size)
    #print("Data size:", data_size)
    #print("Received:", data)

    response = proto.types.Response()
    response.ParseFromString(data)
    return response

class NymProxy:

    def __init__(self, hostname="127.0.0.1", port=9001):
        self._hostname = hostname
        self._port = port

    async def start(self):
        self._reader, self._writer = await asyncio.open_connection(
            self._hostname, self._port)

    async def send(self, message, recipient):
        request = proto.types.Request(
            send=proto.types.RequestSendMessage(
                message=message, recipient=recipient))
        await self._send(request)

        return await self._flush_collect('send')

    async def fetch(self):
        request = proto.types.Request(
            fetch=proto.types.RequestFetchMessages())
        await self._send(request)

        response = await self._flush_collect('fetch')
        return response.fetch.messages

    async def clients(self):
        request = proto.types.Request(
            clients=proto.types.RequestGetClients())
        await self._send(request)

        response = await self._flush_collect('clients')
        return response.clients.clients

    async def details(self):
        request = proto.types.Request(
            details=proto.types.RequestOwnDetails())
        await self._send(request)

        response = await self._flush_collect('details')
        return response.details.details

    async def _do_flush(self):
        flush_request = proto.types.Request(
            flush=proto.types.RequestFlush())
        await self._send(flush_request)

    async def _collect_response(self, request_type, number_responses):
        responses = [await self._read() for i in range(number_responses)]
        #print("Responses:", responses)
        return next(response for response in responses
                    if response.WhichOneof('value') == request_type)

    async def _flush_collect(self, request_type):
        await self._do_flush()
        return await self._collect_response(request_type, 2)

    async def _read(self):
        return await read_response(self._reader)
    async def _send(self, request):
        await send(self._writer, request)

async def run_client():
    nym = NymProxy()
    await nym.start()

    my_details = await nym.details()
    print(my_details)
    await nym.send(b'fooompdd', my_details)
    messages = await nym.fetch()
    print("Messages:", messages)
    print()

    clients = await nym.clients()
    print("Client #4:", clients[4])

    print('Finished')

asyncio.run(run_client())

