# WhatsApp: recordatorios, bandeja y plantillas

Checklist para poner en marcha WhatsApp en Pengi. Cada clínica (tenant) usa su
propio número para:
- enviar recordatorios de citas;
- responder a sus pacientes;
- iniciar conversaciones con plantillas.

Ve marcando las casillas `[ ]` → `[x]` a medida que avances.

> Estado (2026-10-06): código en la rama `feat/whatsapp`, ya subida. Falta:
> - que Meta apruebe las plantillas, para la prueba de punta a punta de los
>   recordatorios;
> - la configuración de producción en Meta (sección 4);
> - decidir lo de los límites de plan antes del deploy (sección 4, paso 0).

---

## 1. Qué hace

- **Conexión por clínica.** En **Ajustes → Integraciones → WhatsApp** hay dos
  formas de conectar:
  - el botón **Conectar WhatsApp** (Embedded Signup de Meta, un popup tipo OAuth);
  - **Conectar manualmente**, con WABA ID, Phone Number ID y un token.
- **Plantillas automáticas.** Al conectar se crean 5 plantillas UTILITY en `es`.
  El detalle está en la sección 8.1.
- **Recordatorios.**
  - Hasta 3 antelaciones por clínica: 48, 24, 12, 2 o 1 h. Las cuentas nuevas
    empiezan con 24 h.
  - Se envían a las citas `scheduled` o `confirmed` de pacientes que **aceptaron
    recibir WhatsApp** (`whatsapp_opt_in`; ver la sección 8.3). Meta exige ese
    consentimiento.
  - No se envía nada mientras la plantilla `pengi_cita_recordatorio` no esté
    **APPROVED**, ni cuando la clínica llegó a su tope mensual (sección 8.2).
  - Nunca se repite un recordatorio: índice único `(appointment_id, offset_hours)`.
- **Respuestas del paciente.**
  - **Confirmar** pasa la cita a `confirmed`, solo si estaba `scheduled`.
  - **Cancelar** la pasa a `cancelled`, si estaba `scheduled` o `confirmed`.
  - En ambos casos se sincroniza con Google Calendar si está conectado y llega
    una notificación in-app a la clínica.
  - Solo cuenta si la respuesta viene del teléfono al que se envió el
    recordatorio y, cuando Meta lo indica, si cita ese mismo mensaje.
- **Baja del paciente.**
  - Si responde `STOP`, `BAJA`, `PARAR`, `DETENER`, `DARME DE BAJA` o
    `NO ENVIAR`, se le quita el consentimiento en esa clínica (origen
    `whatsapp_stop`) y no recibe más recordatorios.
  - `NO` y `CANCELAR` no cuentan: se refieren a la cita.
- **Bandeja de conversaciones:** módulo **WhatsApp** en el menú (sección 7).
- **Estado nuevo de cita.** `confirmed` aparece en la sala de espera, en la
  agenda del día, en el detalle de la cita y en la columna de próxima cita.
- **Seguridad.**
  - El token y el PIN 2FA se guardan cifrados con `secretbox`, nunca se
    devuelven en JSON y nunca se registran en logs.
  - El webhook valida la firma `X-Hub-Signature-256` y rechaza las peticiones
    sin firma válida.
- **Permisos** (categoría `WHATSAPP`, que activa el flag
  `enabled_features.whatsapp`):
  - `MANAGE_WHATSAPP`: Ajustes (conexión, recordatorios, plantillas). Lo tiene
    admin.
  - `USE_WHATSAPP_INBOX`: bandeja, "Nuevo mensaje" y envíos. Lo tienen admin,
    recepcionista y doctor.

---

## 2. Antes de empezar

- [ ] **Regenerar el access token de prueba** en el panel de Meta. Quedó expuesto
      en un chat. Nunca escribas tokens, Phone Number IDs ni WABA IDs en archivos
      del repo.
- [ ] Ten en cuenta que la base local **ya tiene aplicadas** las migraciones
      `DB20261005_1` a `DB20261005_6`: el contenedor de dev se recargó durante el
      desarrollo.
- [ ] Si una base de dev quedó con el índice `idx_whatsapp_messages_wam_id`
      **único** (de una versión intermedia), recréalo normal. Si no, el segundo
      envío fallido da 500:
      ```sql
      DROP INDEX IF EXISTS idx_whatsapp_messages_wam_id;
      CREATE INDEX idx_whatsapp_messages_wam_id ON whatsapp_messages (wam_id);
      ```

---

## 3. Probar en local (modo manual + número de prueba)

### 3.1 Token permanente

El token temporal del panel caduca en unas 24 h y entonces los envíos fallan con
el error 190.

- [ ] Ve a business.facebook.com → **Configuración → Usuarios del sistema** y crea
      un usuario del sistema con rol **Admin**.
- [ ] En **Asignar activos**, dale control total de la app y de la cuenta de
      WhatsApp.
- [ ] Pulsa **Generar token** con:
  - caducidad **Nunca**;
  - permisos `business_management`, `whatsapp_business_messaging` y
    `whatsapp_business_management`.
- [ ] Guárdalo en tu gestor de contraseñas, no en el repo.

### 3.2 Variables de entorno (`apps/api/.env`)

- [ ] `META_APP_ID`: el ID de la app en developers.facebook.com.
- [ ] `META_APP_SECRET`: el *App secret* de la app. Sin él, el webhook rechaza
      todo.
- [ ] `WHATSAPP_WEBHOOK_VERIFY_TOKEN`: un string aleatorio que inventas tú
      (`openssl rand -hex 16`).
- [ ] `WHATSAPP_ENCRYPTION_KEY`: `openssl rand -base64 32`. Es opcional: si
      falta, se usa `SIGNATURE_ENCRYPTION_KEY`.
- [ ] `WHATSAPP_GRAPH_VERSION`: déjala en `v23.0` salvo que Meta la depreque.
- [ ] `META_ES_CONFIG_ID`: déjala vacía por ahora. Se llena en la sección 4.
- [ ] Aplica los cambios con `docker compose -f docker-compose.dev.yaml up -d api`.
      **`restart` no vuelve a leer el `.env`**: hay que recrear el contenedor.

### 3.3 Exponer el webhook (túnel fijo con ngrok, una sola vez)

El servicio `tunnel` de `docker-compose.dev.yaml` levanta ngrok con un **dominio
estático**: la URL nunca cambia y arranca sola con `just dev`.

- [ ] Reclama tu dominio estático gratis en https://dashboard.ngrok.com/domains.
      Algo como `foo-bar.ngrok-free.app`.
- [ ] Copia tu authtoken desde https://dashboard.ngrok.com/get-started/your-authtoken.
- [ ] Crea el `.env` en la raíz del repo (`cp .env.example .env`, está en el
      gitignore) con:
      ```
      COMPOSE_PROFILES=tunnel
      NGROK_AUTHTOKEN=<tu authtoken>
      NGROK_DOMAIN=foo-bar.ngrok-free.app
      ```
- [ ] Ejecuta `just dev`. El túnel queda en el contenedor `pengi-tunnel`.
      - El inspector de peticiones está en http://localhost:4040.
      - Si el stack ya estaba arriba sin el perfil, basta con `just tunnel`.
- [ ] Comprueba que responde: `curl https://<NGROK_DOMAIN>/health`.

### 3.4 Configurar el webhook en Meta

En el panel nuevo de Meta, WhatsApp está dentro del caso de uso *"Connect with
customers through WhatsApp"*, en el menú izquierdo con el ícono del lápiz.

- [ ] Abre tu app → caso de uso de WhatsApp → **Configuration**.
- [ ] Callback URL: `https://<NGROK_DOMAIN>/api/v1/webhooks/whatsapp`.
- [ ] Verify token: el mismo valor que `WHATSAPP_WEBHOOK_VERIFY_TOKEN`.
- [ ] Pulsa **Verify and save**. Debe quedar en verde.
- [ ] Suscribe los campos `messages` y `message_template_status_update`.

> Con el dominio estático esto se configura una sola vez. Cuando el stack está
> apagado, Meta reintenta los eventos durante un tiempo y luego los descarta.
> En producción la URL a nivel de app apunta a producción, y el número de prueba
> se queda en ngrok con `override_callback_uri` (sección 4).

### 3.5 Lista de destinatarios permitidos

El número de prueba solo puede escribir a números de esa lista, máximo 5.

- [ ] Ve a tu app → caso de uso de WhatsApp → **API Testing**.
- [ ] En **To → Manage phone number list**, agrega tu celular con `+593…` y
      confírmalo con el código.
- [ ] Si no está en la lista, el envío falla con el error 131030.

### 3.6 Permisos, plan y tope

- [ ] En el **backoffice**, el plan del tenant de prueba debe incluir el feature
      **WHATSAPP**, que trae `MANAGE_WHATSAPP` y `USE_WHATSAPP_INBOX`.
- [ ] En **Límites del plan**, revisa **Mensajes de WhatsApp**
      (`max_whatsapp_messages`, 300 por defecto).
- [ ] Recarga la app web. Debe aparecer:
  - la tarjeta de WhatsApp en Ajustes → Integraciones;
  - el ítem **WhatsApp** en el menú.

### 3.7 Conectar y probar los recordatorios

- [ ] En Ajustes → Integraciones → WhatsApp → **Conectar manualmente**, ingresa:
  - el WABA ID y el Phone Number ID del número de prueba (están en API Testing);
  - el token permanente del paso 3.1.
- [ ] Comprueba que la tarjeta muestra:
  - el número;
  - las 5 plantillas en `PENDING`;
  - la barra de uso en `0 / 300`.
- [ ] Espera a que `pengi_cita_recordatorio` quede **APPROVED**. Suele tardar
      minutos, a veces horas. Si no se actualiza sola, pulsa **Sincronizar**.
- [ ] **Enviar prueba** a tu celular. Debe llegar el mensaje con los botones, y la
      barra de uso sube a 1.
- [ ] El paciente de prueba debe tener tu celular y la casilla **"Acepta recibir
      recordatorios por WhatsApp"** marcada. Sin ella no se envía nada.
- [ ] Crea una cita para ese paciente **dentro de las próximas 24 h** (por
      ejemplo, en 2–3 horas) con la antelación de 24 h activa.
      - No hace falta esperar: si la cita ya está dentro de la ventana, el
        recordatorio sale en el siguiente minuto.
- [ ] Toca **Confirmar** y verifica:
  - la cita aparece como `confirmed`;
  - llega la notificación in-app;
  - el hilo muestra "✓ Confirmó".
- [ ] Repite con otra cita y toca **Cancelar**: la cita debe quedar `cancelled`.
- [ ] Escribe `STOP` al número: la casilla del paciente debe quedar desmarcada y
      no deben salir más recordatorios para él.

---

## 4. Producción (que cada clínica conecte su número con el botón)

Esto es lo que más tarda, así que conviene empezarlo pronto.

- [ ] **0. Límites de plan (antes del deploy).**
  - Este cambio corrige `GetPlanLimitForCompany`, que ignoraba todos los
    límites guardados en la base. Con el arreglo, `max_patients` y `max_users`
    **empiezan a aplicarse**.
  - Antes de desplegar, revisa en el backoffice el límite de cada plan contra
    cuántos pacientes y usuarios tiene cada clínica.
  - Una clínica que ya está por encima del límite no podrá crear más hasta que
    se suba el límite o se cambie de plan. Los datos existentes no se tocan.
- [ ] **Tech Provider.** Registra a Pengi Studio como Tech Provider de WhatsApp
      en el App Dashboard. El negocio ya está verificado.
- [ ] **App Review** de `whatsapp_business_messaging` y
      `whatsapp_business_management` (acceso avanzado). Meta pide:
  - un video del flujo (conectar en Ajustes → recordatorio → confirmar →
    bandeja);
  - una descripción del uso.
- [ ] **Embedded Signup.** En el App Dashboard → **Facebook Login for Business →
      Configurations**, crea una configuración de tipo *WhatsApp Embedded Signup*.
      Copia el `config_id` en `META_ES_CONFIG_ID`.
- [ ] Agrega el dominio de producción a los dominios permitidos de la app (SDK de
      JS de Facebook).
- [ ] Pon la app en modo **Live**.
- [ ] **Pago:** Pengi Studio paga los mensajes de plantilla. Configura el método de
      pago o la línea de crédito de Tech Provider en Meta.
- [ ] En el servidor de producción:
  - [ ] agrega las variables del paso 3.2 (`deploy/.env.prod.example`) con
        valores reales. El verify token y el `WHATSAPP_ENCRYPTION_KEY` deben ser
        **nuevos**: no reutilices los de desarrollo;
  - [ ] cambia la Callback URL **de la app** a
        `https://api.gentoo.pengistudio.com/api/v1/webhooks/whatsapp`;
  - [ ] para seguir probando en local, apunta solo la cuenta del número de prueba
        a ngrok con `override_callback_uri` (`POST /{waba_id}/subscribed_apps`).
- [ ] Antes del deploy:
  - saca un backup (`pg_dump`) y etiqueta las imágenes anteriores
    (`pengi-*:pre-deploy-YYYYMMDD`);
  - ten en cuenta que hay 6 migraciones nuevas (`DB20261005_1..6`).
- [ ] Revisa en el backoffice que los planes con WhatsApp tengan el tope de
      mensajes que quieres. La migración pone 300 a los que no lo tenían.
- [ ] Prueba con una clínica real: **Conectar WhatsApp** → popup de Meta → elige o
      crea el número → la tarjeta debe quedar conectada con las plantillas en
      `PENDING`.

---

## 5. Referencia técnica

**Backend** (`apps/api`)

| Qué | Dónde |
|---|---|
| Cliente Graph API, firma del webhook | `core/whatsapp/client.go`, `core/whatsapp/webhook.go` |
| Modelos (`WhatsAppAccount`, `WhatsAppMessage`, `WhatsAppConversation`, `WhatsAppTemplate`) | `features/whatsapp/models/` |
| Teléfono a E.164, catálogo de plantillas, agenda, consumer de la cola | `features/whatsapp/services/` (`phone.go`, `templates.go`, `schedule.go`, `sender.go`) |
| Conversaciones, uso mensual, consentimiento, avisos | `services/conversations.go`, `usage.go`, `consent.go`, `notifications.go` |
| Scheduler (cada 1 min, recupera mensajes trabados) | `features/whatsapp/workers/reminder-scheduler.go` |
| Handlers + webhook + flujo de consentimiento | `features/whatsapp/handlers/` |
| Rutas | `routes/whatsapp_routes.go`; marcado en lote en `routes/clinical_routes.go` |
| Migraciones | `migrations/code-migrations/2026/add_whatsapp_permissions.go` (`_1`, `_2`), `add_whatsapp_inbox.go` (`_3`, `_4`), `add_whatsapp_templates_and_limit.go` (`_5`, `_6`) |
| Límites de plan | `features/companies/middleware/plan-limit.go` |
| Códigos de error | `core/errors/codes.go` (`E-WA-001…023`) |

**Endpoints** (`/api/v1/whatsapp`)

| Método | Ruta | Permiso | Uso |
|---|---|---|---|
| GET | `/config` | MANAGE | `app_id` y `config_id` para el SDK |
| GET | `/account` | MANAGE | Estado de la conexión y configuración |
| POST | `/connect/manual` · `/connect/embedded` | MANAGE | Conectar |
| POST | `/template/sync` | MANAGE | Vuelve a consultar o crea las 5 plantillas |
| PUT | `/settings` | MANAGE | `{reminders_enabled, reminder_offsets}`: entre 1 y 168 h, máximo 3 |
| DELETE | `/account` | MANAGE | Desconecta; el historial se conserva |
| POST | `/test` | MANAGE | `{phone}`: plantilla de ejemplo (consume cupo) |
| GET | `/usage` | MANAGE o INBOX | `{used, limit, period_start, period_end}`; `limit` -1 significa ilimitado |
| GET | `/messages` | MANAGE o INBOX | Envíos (`page`, `limit`, `status`, `kind`, `appointment_id`) |
| GET | `/templates` | INBOX | Catálogo con estado, variables, vista previa y `usable` |
| GET | `/conversations` · `/conversations/unread-count` · `/conversations/:id` | INBOX | Bandeja |
| GET | `/conversations/:id/messages?before=` | INBOX | Hilo paginado |
| POST | `/conversations/:id/messages` | INBOX | Responder con texto libre (ventana de 24 h) |
| POST | `/conversations/start` · `/conversations/:id/template` | INBOX | Enviar plantilla (`{patient_id \| -, template, appointment_id?}`) |
| POST | `/conversations/:id/read` | INBOX | Marca como leídos la conversación y sus avisos |
| PUT | `/conversations/:id/patient` | INBOX | Asocia un número desconocido |

Marcado en lote: `POST /api/v1/clinical/patients/whatsapp-opt-in`
`{ids, opt_in}` (`UPDATE_PATIENT`, máximo 1000) → `{updated, skipped_opted_out}`.

El webhook es público:
- `GET /api/v1/webhooks/whatsapp` hace el handshake.
- `POST /api/v1/webhooks/whatsapp` recibe los eventos.

**Mensajes**
- **Tipos (`kind`):**
  - `reminder`, `test` y `template`: cuentan para el tope;
  - `reply`: respuesta con texto libre, gratis;
  - `inbound`: mensaje del paciente;
  - `system`: pregunta de consentimiento, no cuenta.
- **Estados salientes:** `queued → sending → sent → delivered → read → replied`.
  Además:
  - `failed`: error de Meta o número inválido;
  - `skipped`: la cita se canceló o ya empezó, el paciente retiró el
    consentimiento (`no_opt_in`) o se llegó al tope (`monthly_limit`).
- **Estado entrante:** `received`.

**Errores frecuentes de Meta** (se muestran traducidos en el hilo y en Envíos)

| Código | Significa |
|---|---|
| 131026 | El número no tiene WhatsApp o no puede recibir |
| 131030 | El destinatario no está en la lista permitida del número de prueba |
| 131047 | Pasaron más de 24 h desde el último mensaje del cliente (no aplica a plantillas) |
| 131049 | Meta limitó el envío para cuidar la experiencia del usuario |
| 132001 | La plantilla no existe o no está aprobada en ese idioma |
| 190 | Token inválido o caducado: reconectar con un token permanente |

**Frontend** (`apps/web/src`)

| Qué | Dónde |
|---|---|
| Service + tipos | `api/whatsapp-service.ts`; marcado en lote en `api/clinical-service.ts` |
| Tarjeta de ajustes, plantillas y uso | `sections/settings/whatsapp-settings.tsx`, `whatsapp-templates-usage.tsx` |
| Embedded Signup (hook + SDK) | `hooks/use-embedded-signup.ts`, `lib/facebook-sdk.ts` |
| Módulo WhatsApp | `pages/whatsapp/whatsapp-page.tsx`, `sections/whatsapp/` |
| "Nuevo mensaje" / "Enviar plantilla" | `sections/whatsapp/template-message-dialog.tsx` |
| Escribir desde la ficha | `components/features/whatsapp/open-chat-button.tsx` |
| Avisos (toast + sonido) | `hooks/use-whatsapp-message-alerts.ts`, `lib/notification-sound.ts` |
| Contador de no leídos | `store/whatsapp-store.ts`, `sections/template/dashboard-template.tsx` |

Backoffice: límite `max_whatsapp_messages` en
`apps/backoffice/src/pages/plans/plan-limits-editor.tsx`.

---

## 6. Riesgos conocidos y pendientes

- **Duplicado raro:** si el proceso cae justo después de que Meta aceptó un
  mensaje, la recuperación (a los 15 min en `sending`) puede reenviarlo.
- **Tope con envíos simultáneos:** puede pasarse por uno o dos mensajes. Se
  acepta.
- **Fallo al contar el uso:** si el conteo falla, los recordatorios salen igual
  (queda en el log) para no cortar el servicio.
- **Salir a mitad de la conexión:** si el usuario sale de Ajustes mientras se
  completa, el servidor la termina igual, pero la UI no se refresca hasta recargar.
- **Teléfonos de pacientes:** son texto libre. Se normalizan a E.164 asumiendo
  Ecuador (`0…` → `593…`). Los números inválidos quedan `failed` con
  `invalid_phone`.
- **Pregunta de consentimiento:** si su envío falla, cuenta igual como
  preguntada y no se reintenta.
- **Las citas no tienen doctor asignado**, así que el recordatorio no lo menciona.
- **Ajustes sin permiso de bandeja:** un usuario con `MANAGE_WHATSAPP` pero sin
  `USE_WHATSAPP_INBOX` solo ve la plantilla del recordatorio.

---

## 7. Bandeja de conversaciones (módulo WhatsApp)

Está en el menú lateral → **WhatsApp**, con un contador de conversaciones sin leer
que se actualiza cada 15 s. Requiere `USE_WHATSAPP_INBOX`.

- **Conversaciones**: una por teléfono y clínica.
  - Se asocia al paciente por su teléfono. Si no hay coincidencia, o hay más de
    una, se usa "Asociar a paciente".
  - El hilo junta recordatorios, plantillas, respuestas con botón, mensajes del
    sistema y textos del paciente, con su estado.
- **Responder**: con texto libre solo dentro de las **24 h** siguientes al último
  mensaje del paciente, por regla de Meta. Pasado ese plazo el compositor ofrece
  **"Enviar plantilla"** (sección 8.1) y el backend rechaza el texto libre con 409
  (`E-WA-017`).
- **Envíos**: la tabla de mensajes salientes, con filtros por estado y tipo.
- **Escribir por WhatsApp**: botón en la ficha del paciente. Abre la
  conversación o el diálogo "Nuevo mensaje" con el paciente ya elegido.
- **Actualización**: hilo cada 5 s, lista cada 15 s. Se pausa con la pestaña
  oculta.
- **Adjuntos**: en la v1 solo se muestran como "[Imagen]", "[Audio]" y similares.

**Probarla con el número de prueba:**
- [ ] Escribe "hola" desde tu celular al +1 555 139 3478. Debe aparecer la
      conversación con 1 sin leer, y el contador del menú sube.
- [ ] Ábrela: el contador baja.
- [ ] Responde desde Pengi: debe llegarte al celular, con los checks de
      entregado y leído.
- [ ] Toca Confirmar en un recordatorio: debe verse en el mismo hilo.

---

## 8. Plantillas, tope mensual, consentimiento y avisos

### 8.1 Plantillas y "Nuevo mensaje"

Pengi crea las 5 plantillas en cada clínica al conectar (o con **Sincronizar**).
Cada una tiene su estado en la tabla `whatsapp_templates`, que el webhook
`message_template_status_update` mantiene al día.

| Plantilla | Para qué | Variables |
|---|---|---|
| `pengi_cita_recordatorio` | Recordatorio automático (Confirmar / Cancelar) | paciente, clínica, fecha, hora |
| `pengi_continuar_conversacion` | Reabrir una conversación | paciente, clínica |
| `pengi_resultados_listos` | Avisar que hay resultados | paciente, clínica |
| `pengi_reprogramar_cita` | Coordinar otro horario (pide elegir la cita) | paciente, clínica, fecha, hora |
| `pengi_control_seguimiento` | Recordar agendar un control | paciente, clínica |

- **Nuevo mensaje**, en la bandeja:
  1. elegir paciente; solo aparecen los que tienen teléfono, y avisa si no tiene
     consentimiento;
  2. elegir plantilla, con vista previa;
  3. elegir la cita si es "reprogramar";
  4. enviar.
- Solo se pueden usar plantillas **APPROVED** y solo con pacientes con
  consentimiento.
- Cuando el paciente responde, se abre la ventana de 24 h y se sigue con texto
  libre.

### 8.2 Tope mensual por plan

- **Límite:** `max_whatsapp_messages` en el plan. Se edita en el backoffice;
  ilimitado = -1.
- **Qué cuenta:** recordatorios, pruebas y plantillas que llegaron a Meta en el
  **mes calendario**, en hora de Ecuador. Las respuestas (`reply`) y los
  mensajes del sistema no cuentan.
- **Al llegar al tope:**
  - los recordatorios quedan `skipped` con `monthly_limit`;
  - "Nuevo mensaje", "Enviar plantilla" y "Enviar prueba" responden 403
    (`E-WA-023`).
- **Avisos:** al **80 %** y al **100 %**, notificación a quienes tienen
  `MANAGE_WHATSAPP`, una vez por mes.
- **Ajustes** muestra la barra "Mensajes este mes: X / Y".

### 8.3 Consentimiento de los pacientes

| Cómo se marca | Origen (`whatsapp_opt_in_source`) |
|---|---|
| Al crear el paciente: la casilla viene marcada por defecto | `registration` |
| Al editar la ficha | `manual` |
| En lote: Pacientes → seleccionar → **Consentimiento WhatsApp** | `bulk` |
| El paciente responde **SÍ** a la pregunta por WhatsApp | `whatsapp` |
| El paciente responde **STOP** (baja) | `whatsapp_stop` |

- **El marcado en lote omite a quienes respondieron STOP.** Para volver a
  suscribirlos hay que hacerlo uno por uno, desde su ficha. El aviso tras el
  marcado en lote lo explica.
- **Pregunta por WhatsApp:**
  - Si escribe un paciente sin consentimiento, Pengi le pregunta una sola vez
    por conversación: "responde SÍ para recibir recordatorios".
  - No se pregunta tras un STOP, a números desconocidos ni si los recordatorios
    no se pueden enviar.
  - Un "SÍ" solo cuenta si antes se le preguntó, y marca a todos los pacientes
    con ese número.

### 8.4 Avisos de mensajes nuevos

- **Campana:** una notificación por conversación con mensajes sin leer, no una
  por mensaje. Lleva al chat y se marca como leída al abrirlo.
- **Toast con sonido:** si no estás en la bandeja. El sonido se silencia con el
  interruptor de la bandeja y no suena con la pestaña oculta.

**Probar 8.1–8.4:**
- [ ] Pulsa **Sincronizar** en Ajustes y comprueba que las 5 plantillas aparecen
      con su estado y la barra de uso.
- [ ] Usa **Pacientes → seleccionar → Consentimiento WhatsApp → Marcar**.
      Comprueba que un paciente con STOP no se marca y que el aviso lo explica.
- [ ] Escribe desde un número de paciente sin consentimiento: debe llegar la
      pregunta, y al responder "SÍ" la casilla se marca.
- [ ] Escribe desde tu celular estando en otra página: debe llegar el toast con
      sonido, más la notificación en la campana.
- [ ] Cuando aprueben `pengi_continuar_conversacion`, usa **Nuevo mensaje** →
      paciente → plantilla → enviar: debe llegar, y la barra de uso sube.
- [ ] Baja el tope a 1 en el backoffice y comprueba que "Nuevo mensaje" queda
      bloqueado y que llega el aviso del 100 %.

**Pendiente para después:**
- adjuntos (ver y enviar imágenes y PDFs);
- asignar conversaciones a usuarios;
- respuestas rápidas;
- tiempo real con websockets;
- avisar al admin si el token se invalida o baja la calidad del número;
- tests E2E con Playwright;
- paquetes de mensajes extra.
