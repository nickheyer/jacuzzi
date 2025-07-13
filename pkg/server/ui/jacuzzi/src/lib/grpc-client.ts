import { createClient, type Client } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";
import { TemperatureService } from "./proto/jacuzzi/v1/temperature_pb";

const transport = createGrpcWebTransport({
  baseUrl: window.location.origin,
});

export const temperatureClient: Client<typeof TemperatureService> = createClient(TemperatureService, transport);