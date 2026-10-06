# WhatsApp — recordatorios de citas

Checklist para poner en marcha los recordatorios de citas por WhatsApp. Cada clínica
(tenant) los envía desde su propio número. Ve marcando las casillas `[ ]` → `[x]`
a medida que avances.

> Estado (2026-10-05): el código está implementado y verificado, pero sin commit.
> Falta la configuración en Meta y la prueba de punta a punta.

---

## 1. Qué hace

- **Conexión por clínica.** En **Ajustes → Integraciones → WhatsApp** hay dos
  formas de conectar:
  - el botón **Conectar WhatsApp** (Embedded Signup de Meta, un popup tipo OAuth);
  - **Conectar manualmente**, con WABA ID, Phone Number ID y un token.
- **Plantilla automática.** Al conectar se crea la plantilla `pengi_cita_recordatorio`:
  - categoría UTILITY, idioma `es`;
  - variables: paciente, clínica, fecha y hora;
  - botones **Confirmar** y **Cancelar**.
- **Recordatorios.**
  - Hasta 3 antelaciones por clínica: 48, 24, 12, 2 o 1 h. Las cuentas nuevas
    empiezan con 24 h.
  - Se envían a las citas `scheduled` o `confirmed` de pacientes que **aceptaron
    recibir WhatsApp** (casilla "Acepta recibir recordatorios por WhatsApp" en la
    ficha del paciente, `whatsapp_opt_in`). Meta exige ese consentimiento.
  - No se envía nada mientras la plantilla no esté **APPROVED**.
  - Nunca se repite un recordatorio: índice único `(appointment_id, offset_hours)`.
- **Respuestas del paciente.**
  - **Confirmar** pasa la cita a `confirmed`, solo si estaba `scheduled`.
  - **Cancelar** la pasa a `cancelled`, si estaba `scheduled` o `confirmed`.
  - En ambos casos se sincroniza con Google Calendar si está conectado y llega
    una notificación in-app a todos los usuarios de la clínica.
  - Solo cuenta si la respuesta viene del teléfono al que se envió el
    recordatorio y, cuando Meta lo indica, si cita ese mismo mensaje.
- **Baja del paciente.** Si responde `STOP`, `BAJA`, `PARAR`, `DETENER`,
  `DARME DE BAJA` o `NO ENVIAR`, se le quita el consentimiento en esa clínica y no
  recibe más recordatorios. `NO` y `CANCELAR` no cuentan: se refieren a la cita.
- **Panel.** Muestra el estado de la plantilla e incluye:
  - un botón para volver a consultar la plantilla (`POST /whatsapp/template/sync`);
  - envío de un mensaje de prueba;
  - el historial paginado de mensajes, con su estado y el error traducido.
- **Estado nuevo de cita.** `confirmed` aparece en la sala de espera, en la
  agenda del día, en el detalle de la cita y en la columna de próxima cita.
- **Seguridad.**
  - El token y el PIN 2FA se guardan cifrados con `secretbox`, nunca se
    devuelven en JSON y nunca se registran en logs.
  - El webhook valida la firma `X-Hub-Signature-256` y rechaza las peticiones
    sin firma válida.
- **Permiso.** `MANAGE_WHATSAPP`, de categoría `WHATSAPP`, activa el flag
  `enabled_features.whatsapp`.

---

## 2. Antes de empezar

- [ ] **Regenerar el access token de prueba** en el panel de Meta. Quedó expuesto
      en un chat. Nunca escribas tokens, Phone Number IDs ni WABA IDs en archivos
      del repo.
- [ ] Revisar el diff (`git status`) y hacer commit cuando estés conforme. Nada
      está commiteado.
- [ ] Ten en cuenta que la base local **ya tiene aplicadas** las migraciones
      `DB20261005_1` y `DB20261005_2`: el contenedor de dev se recargó durante el
      desarrollo.

---

## 3. Probar en local (modo manual + número de prueba)

### 3.1 Token permanente

El token temporal del panel caduca en unas 24 h.

- [ ] Meta Business Suite → **Configuración del negocio → Usuarios del sistema**.
      Crea un usuario del sistema con rol admin.
- [ ] Asígnale la app y la cuenta de WhatsApp.
- [ ] Genera un token con los permisos `whatsapp_business_messaging` y
      `whatsapp_business_management`, con caducidad **Nunca**.
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
- [ ] `META_ES_CONFIG_ID`: déjala vacía por ahora. Se llena en el paso 4.
- [ ] Reinicia la API (`just dev`).

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

- [ ] developers.facebook.com → tu app → **WhatsApp → Configuración → Webhook**.
- [ ] Callback URL: `https://<NGROK_DOMAIN>/api/v1/webhooks/whatsapp`.
- [ ] Verify token: el mismo valor que `WHATSAPP_WEBHOOK_VERIFY_TOKEN`.
- [ ] Pulsa **Verificar y guardar**. Debe quedar en verde.
- [ ] Suscribe los campos `messages` y `message_template_status_update`.

> Con el dominio estático esto se configura una sola vez. Cuando el stack está
> apagado, Meta reintenta los eventos durante un tiempo y luego los descarta.

### 3.5 Lista de destinatarios permitidos

El número de prueba solo puede escribir a números de esa lista.

- [ ] developers.facebook.com → tu app → **WhatsApp → Configuración de la API**.
- [ ] En el campo **Para**, agrega tu celular y confírmalo con el código que llega.

### 3.6 Permiso y plan

- [ ] En el **backoffice**, agrega el permiso `MANAGE_WHATSAPP` al plan del
      tenant de prueba. La migración no lo agrega a ningún plan, solo al rol admin.
- [ ] Recarga la app web y comprueba que aparece la tarjeta de WhatsApp en
      Ajustes → Integraciones.

### 3.7 Conectar y probar

- [ ] En Ajustes → Integraciones → WhatsApp → **Conectar manualmente**, ingresa:
  - el WABA ID del número de prueba (está en el panel de la app);
  - su Phone Number ID (también en el panel);
  - el token permanente del paso 3.1.
- [ ] Comprueba que la tarjeta muestra el número y el estado de la plantilla
      (`PENDING`).
- [ ] Espera a que la plantilla quede **APPROVED**. Suele tardar minutos, a veces
      horas. Si no se actualiza sola, pulsa el botón de sincronizar.
- [ ] **Enviar prueba** a tu celular. Debe llegar el mensaje con los botones.
- [ ] Comprueba que el historial muestra el mensaje como `sent → delivered → read`.
- [ ] Marca en la ficha del paciente de prueba la casilla **"Acepta recibir
      recordatorios por WhatsApp"**. Sin ella no se envía nada.
- [ ] Crea una cita para un paciente con **tu celular**, para dentro de
      **24 h + 2 min**. Configura la antelación en 24 h.
- [ ] En 1–3 minutos debe llegar el recordatorio. El scheduler corre cada minuto.
- [ ] Toca **Confirmar** y verifica:
  - la cita aparece como `confirmed`;
  - llega la notificación in-app;
  - el historial muestra `replied` / confirmar.
- [ ] Repite con otra cita y toca **Cancelar**: la cita debe quedar `cancelled`.
- [ ] Escribe `STOP` al número: la casilla del paciente debe quedar desmarcada y
      no deben salir más recordatorios para él.

---

## 4. Producción (que cada clínica conecte su número con el botón)

Esto es lo que más tarda, así que conviene empezarlo pronto.

- [ ] **Tech Provider.** Registra a Pengi Studio como Tech Provider de WhatsApp
      en el App Dashboard. El negocio ya está verificado.
- [ ] **App Review** de `whatsapp_business_messaging` y
      `whatsapp_business_management` (acceso avanzado). Meta pide:
  - un video del flujo (conectar en Ajustes → recordatorio → confirmar);
  - una descripción del uso.
- [ ] **Embedded Signup.** En el App Dashboard → **Facebook Login for Business →
      Configurations**, crea una configuración de tipo *WhatsApp Embedded Signup*.
      Copia el `config_id` en `META_ES_CONFIG_ID`.
- [ ] Agrega el dominio de producción a los dominios permitidos de la app (SDK de
      JS de Facebook).
- [ ] Pon la app en modo **Live**.
- [ ] En el servidor de producción:
  - [ ] agrega las variables del paso 3.2 (`deploy/.env.prod.example`) con valores
        reales y un `WHATSAPP_ENCRYPTION_KEY` propio;
  - [ ] cambia la Callback URL del webhook a
        `https://<api-prod>/api/v1/webhooks/whatsapp`.
- [ ] Antes del deploy, saca un backup (`pg_dump`) y etiqueta la imagen anterior,
      como en cada deploy riesgoso.
- [ ] Prueba con una clínica real: **Conectar WhatsApp** → popup de Meta → elige o
      crea el número → la tarjeta debe quedar conectada con la plantilla en
      `PENDING`.

---

## 5. Referencia técnica

**Backend** (`apps/api`)

| Qué | Dónde |
|---|---|
| Cliente Graph API, firma del webhook | `core/whatsapp/client.go`, `core/whatsapp/webhook.go` |
| Modelos (`WhatsAppAccount`, `WhatsAppMessage`) | `features/whatsapp/models/` |
| Teléfono a E.164, plantilla, agenda, consumer de la cola | `features/whatsapp/services/` (`phone.go`, `templates.go`, `schedule.go`, `sender.go`) |
| Scheduler (cada 1 min, recupera mensajes trabados) | `features/whatsapp/workers/reminder-scheduler.go` |
| Handlers + webhook | `features/whatsapp/handlers/` |
| Rutas | `routes/whatsapp_routes.go`, webhook en `routes/index.go` |
| Migración (permiso + flag) | `migrations/code-migrations/2026/add_whatsapp_permissions.go` |
| Códigos de error | `core/errors/codes.go` (`E-WA-001…014`) |

**Endpoints** (`/api/v1/whatsapp`, requieren `MANAGE_WHATSAPP`)

| Método | Ruta | Uso |
|---|---|---|
| GET | `/config` | `app_id` y `config_id` para el SDK; dice si el embedded signup está disponible |
| GET | `/account` | Estado de la conexión, plantilla y configuración |
| POST | `/connect/manual` | `{waba_id, phone_number_id, access_token}` |
| POST | `/connect/embedded` | `{code, waba_id, phone_number_id}` |
| POST | `/template/sync` | Vuelve a consultar la plantilla o la crea |
| PUT | `/settings` | `{reminders_enabled, reminder_offsets}`: entre 1 y 168 h, máximo 3 |
| DELETE | `/account` | Desconecta; el historial se conserva |
| POST | `/test` | `{phone}`: envía la plantilla con datos de ejemplo |
| GET | `/messages` | Historial paginado (`page`, `limit`, `status`, `appointment_id`) |

El webhook es público:
- `GET /api/v1/webhooks/whatsapp` hace el handshake.
- `POST /api/v1/webhooks/whatsapp` recibe los eventos.

**Estados de mensaje:** `queued → sending → sent → delivered → read → replied`.
Además existen:
- `failed`: error de Meta o número inválido;
- `skipped`: la cita se canceló o ya empezó antes de enviar, o el paciente
  retiró el consentimiento (`no_opt_in`) después de que se encoló.

**Errores frecuentes de Meta** (se muestran traducidos en el historial)

| Código | Significa |
|---|---|
| 131026 | El número no tiene WhatsApp o no puede recibir |
| 131047 | Pasaron más de 24 h desde el último mensaje del cliente (no aplica a plantillas) |
| 131049 | Meta limitó el envío para cuidar la experiencia del usuario |
| 132001 | La plantilla no existe o no está aprobada en ese idioma |
| 190 | Token inválido o caducado: reconectar |

**Frontend** (`apps/web/src`)

| Qué | Dónde |
|---|---|
| Service + tipos | `api/whatsapp-service.ts` |
| Tarjeta de ajustes | `sections/settings/whatsapp-settings.tsx` |
| Embedded Signup (hook + SDK) | `hooks/use-embedded-signup.ts`, `lib/facebook-sdk.ts` |
| Diálogos | `components/features/whatsapp/` |
| Columnas del historial | `sections/columns/whatsapp/whatsapp-message-columns.tsx` |

---

## 6. Riesgos conocidos y pendientes

- **Duplicado raro:** si el proceso cae justo después de que Meta aceptó un
  mensaje, la recuperación (a los 15 min en `sending`) puede reenviarlo.
- **Salir a mitad de la conexión:** si el usuario sale de Ajustes mientras se
  completa, el servidor la termina igual, pero la UI no se refresca hasta recargar.
- **Teléfonos de pacientes:** son texto libre. Se normalizan a E.164 asumiendo
  Ecuador (`0…` → `593…`). Los números inválidos quedan `failed` con
  `invalid_phone`.
- **Pacientes existentes:** empiezan sin consentimiento. La clínica tiene que
  marcar la casilla en cada ficha cuando el paciente acepte.
- **Las citas no tienen doctor asignado**, así que el recordatorio no lo menciona.
- **Ideas para después:**
  - bandeja de chat con pacientes (ventana de 24 h);
  - recordatorios de otros tipos (resultados de exámenes, controles);
  - filtro por estado `confirmed` en el calendario.

---

## 7. Bandeja de conversaciones (módulo WhatsApp)

Está en el menú lateral → **WhatsApp**, con un contador de conversaciones sin leer.
Requiere el permiso `USE_WHATSAPP_INBOX`, que tienen admin, recepcionista y
doctor. Viene incluido en los planes que tienen el feature WHATSAPP.

- **Conversaciones**: una por teléfono y clínica.
  - Se asocia al paciente por su teléfono. Si no hay coincidencia, o hay más de
    una, se usa "Asociar a paciente".
  - El hilo junta recordatorios, respuestas con botón y textos del paciente,
    con su estado.
- **Responder**: con texto libre solo dentro de las **24 h** siguientes al último
  mensaje del paciente, por regla de Meta. Pasado ese plazo el compositor se
  bloquea y el backend responde 409 (`E-WA-017`).
- **Envíos**: la tabla de mensajes enviados, que antes estaba en Ajustes.
- **Abrir chat**: botón en la ficha del paciente cuando existe una conversación.
- **Actualización**: hilo cada 5 s, lista cada 15 s y contador cada 30 s. Se
  pausa con la pestaña oculta.
- **Adjuntos**: en la v1 solo se muestran como "[Imagen]", "[Audio]" y similares.

**Probarla con el número de prueba:**
- [ ] Escribe "hola" desde tu celular al +1 555 139 3478. Debe aparecer la
      conversación con 1 sin leer, y el contador del menú sube.
- [ ] Ábrela: el contador baja.
- [ ] Responde desde Pengi: debe llegarte al celular, con los checks de
      entregado y leído.
- [ ] Toca Confirmar en un recordatorio: debe verse en el mismo hilo.

**Pendiente para después:**
- adjuntos;
- asignar conversaciones a usuarios;
- respuestas rápidas;
- plantillas para reabrir conversaciones fuera de la ventana;
- tiempo real con websockets;
- definir quién paga los mensajes de plantilla.
