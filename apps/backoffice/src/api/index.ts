import { session } from "@/lib/session";
import { api, noAuthApi } from "./http-clients";

export { api, noAuthApi };

session.attach(api);
