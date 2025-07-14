import { createClient, type Client } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";
import { TemperatureService, ClientService, AlertService, SettingsService } from "./proto/jacuzzi/v1/services_pb";

const transport = createGrpcWebTransport({
  baseUrl: window.location.origin,
});

export const temperatureClient: Client<typeof TemperatureService> = createClient(TemperatureService, transport);
export const clientClient: Client<typeof ClientService> = createClient(ClientService, transport);
export const alertClient: Client<typeof AlertService> = createClient(AlertService, transport);
export const settingsClient: Client<typeof SettingsService> = createClient(SettingsService, transport);