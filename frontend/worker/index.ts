/**
 * ServicePro edge Worker entry point. See ./proxy.ts for the routing and
 * proxy logic. Only the default export lives here: the Workers runtime treats
 * every named export of the main module as an entrypoint.
 */
import { handleRequest } from './proxy';

export default {
  fetch(request, env): Promise<Response> {
    return handleRequest(request, env);
  },
} satisfies ExportedHandler<Env>;
