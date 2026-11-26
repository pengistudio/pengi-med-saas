# Bloqueos de WhatsApp al conectar: hechos y plan de pruebas

Estado al 2026-11-24. La causa de los bloqueos **sigue sin confirmarse** (ver la
actualización del 2026-11-24, al final de las hipótesis). Este
documento separa lo observado de las hipótesis y define pruebas que cambian
una sola variable cada vez.

## Incidentes

| # | Número | Portafolio dueño | Antes del bloqueo | Resultado |
|---|---|---|---|---|
| 1 | Real de una clínica | Sin verificar (cuenta de Facebook creada ese día) | 2 conexiones por coexistence el mismo día, 0 mensajes | "Cuenta en revisión" a los ~3 min |
| 2 | +593 98 830 9269 (prueba; WhatsApp Business creado ese día) | "Walter" (creado ese día, sin verificar, sin método de pago) | Conexión coexistence 15:00, error 10 de Graph al crear plantillas, 0 mensajes | Baneado al ~1 min; restaurado tras revisión |
| 3 | El mismo | "Walter" | Sin reconectar: 1 intento fallido de crear `portal_code` (17:31), 2 mensajes de texto al número personal del dueño (~21:59) | Número Banned; portafolio deshabilitado "Permanent" |

Texto de Meta (Business Support Home): "We have disabled Walter and its
WhatsApp Business accounts due to activity that does not comply with
WhatsApp's Business Terms of Service: breach of Terms of Acceptable Use". El
banner de WhatsApp Manager cita la Commerce Policy. Revisión solicitada el
2026-11-16 (24-48 h).

## Descartado con datos

- Permisos de la app Gentoo (1390004093049552): live, App Review aprobado,
  `whatsapp_business_management` y `whatsapp_business_messaging` con acceso
  avanzado; webhook suscrito (incluye `message_template_status_update`).
- Token guardado: SYSTEM_USER válido con ambos permisos sobre la WABA.
- Spam o plantillas sin aprobar: 0 mensajes con plantilla en total.
- Errores de Gentoo en el incidente 3: 0 líneas warn/error en 24 h.

## Hipótesis (ninguna probada)

- **H1 Portafolio/cuenta de Facebook nuevos**: creados el mismo día, sin
  verificar, sin datos de negocio ni método de pago.
- **H2 Flujo coexistence**: enlazar a la API un número de la app WhatsApp
  Business (común a los tres incidentes).
- **H3 Política de comercio / sector salud**: la Commerce Policy restringe
  productos médicos, medicamentos y telemedicina.
- **H4 Algo que hace Gentoo al conectar** (suscripción, importación de
  historial, llamadas de plantillas). Sin evidencia. La importación de historial
  ya no es sospechosa: nunca se pidió (ver la actualización).
- **H5 Restricciones de Meta a Tech Providers** alrededor de coexistence /
  Embedded Signup, ajenas a Gentoo (añadida el 2026-11-24).

## Actualización 2026-11-24

1. **La causa sigue sin confirmarse.** Nada de esta sección la prueba, y los
   cambios del punto 7 no pretenden arreglar los bloqueos.
2. **H5.** En un hilo de la comunidad de Meta
   (<https://developers.facebook.com/community/threads/891379180714484/>) varios
   Tech Providers, entre ellos de LATAM y Brasil, cuentan restricciones alrededor
   de coexistence / Embedded Signup desde ~17 jul 2026. Algunas apelaciones se
   aceptaron y la cuenta volvió a restringirse. Seguían llegando reportes el 19 y
   el 22 de noviembre. Son testimonios: Meta no ha dado una causa raíz.
3. **Antigüedad del número (parte de H1).** Meta tiene códigos de elegibilidad en
   Embedded Signup: 3441045 ("More activity on the WhatsApp Business App is
   needed to help determine eligibility") y 3441077 ("This number isn't eligible
   yet. Keep using it in the WhatsApp Business app and try again later"), en
   <https://developers.facebook.com/documentation/business-messaging/whatsapp/embedded-signup/errors>.
   360dialog (un proveedor, no Meta) pide 7 días de uso como mínimo y recomienda
   30. Nuestro número no fue rechazado en el alta, así que no es concluyente.
4. **H3 pierde peso.** La entrada de la violación HEALTHCARE de Meta dice
   "Manufacturers and healthcare services who do not engage in direct sale are
   allowed".
5. **H4: la importación de historial queda descartada.** Gentoo nunca llamó a
   `smb_app_data`, el paso que la pide, así que nunca se envió historial.
6. **Revisión automática tras el alta.** Según el changelog de Meta de abril de
   2022, la revisión de cumplimiento se dispara sola, en segundo plano, después
   del alta. Encaja con bloqueos a los pocos minutos, pero no dice el motivo.
7. **Huecos de Gentoo, corregidos en la rama
   `fix/whatsapp-enforcement-events`:**
   - **A. Los avisos de Meta se perdían.** El webhook ignoraba casi todos los
     `account_update` (incluidos `ACCOUNT_VIOLATION`, `DISABLED_UPDATE` y
     `ACCOUNT_RESTRICTION`) y los logs del contenedor se borran en cada deploy.
     Ahora todo `account_update`, `account_review_update` y `account_alerts`
     queda en la tabla `whatsapp_account_events`, aunque la WABA no esté
     conectada todavía.
   - **B. Faltaba un paso obligatorio de coexistence.** Meta exige pedir los
     contactos y el historial (`smb_app_data`) en las 24 h siguientes al alta; si
     no, hay que dar de baja a la clínica y repetir el alta. Ahora se piden al
     conectar, una sola vez y sin reintentos.
   - Campos `account_review_update` y `account_alerts` del webhook activados
     en el App Dashboard de Meta el 2026-11-25 (comprobado por API: la app
     recibe los 8 campos y la URL de callback es la de producción).
8. **Prueba 0, estado.** El 2026-11-24 la API de Meta muestra la cuenta de
   mensajería "Approved" y el número registrado; el portafolio sigue sin
   verificar. Eso **no prueba** que se levantara el bloqueo: hay que mirarlo en
   Business Support Home.
9. **Prueba A, dos avisos.**
   - Error 3441128 de Meta: una cuenta de WhatsApp Business que está en el
     portafolio dueño de la app no se puede conectar por Embedded Signup. Pengi
     Studio es ese portafolio, así que la prueba A tal como estaba escrita
     necesita **otro portafolio verificado**.
   - El cambio B añade una variable a las pruebas futuras: las llamadas a
     `smb_app_data`. Los incidentes 1 a 3 ocurrieron sin ellas, de modo que un
     resultado distinto puede deberse a eso y no solo a lo que la prueba aísla.

## Reglas para toda prueba

- Solo números de repuesto. Nunca el número real de una clínica.
- Un solo intento de conexión por número. Sin desconectar ni reconectar.
- Producción con #134, #135 y #136 desplegados (nada crea plantillas solo).
- Antes de empezar: la rama `fix/whatsapp-enforcement-events` desplegada y los
  campos `account_review_update` y `account_alerts` suscritos en el App Dashboard.
  Sin eso se vuelve a perder el motivo que dé Meta.
- Solo números con 7 días o más de uso real en WhatsApp Business (ideal: 30). En
  la prueba B, que usa un número sin app, no aplica.
- No pulsar "Crear plantillas faltantes" ni enviar mensajes hasta el paso
  indicado. Esperar 24 h entre pasos.
- Registrar en cada paso: hora, estado del número (WhatsApp Manager), estado
  del portafolio (Business Support Home), logs de la API (`warn`/`error`) y las
  filas nuevas de `whatsapp_account_events`.

## Pruebas

### Prueba 0 — Resultado de la revisión de "Walter"
- Restaurado → falso positivo automático; añadir método de pago y completar
  datos del negocio antes de cualquier otra prueba con ese portafolio.
- Rechazado → anotar el motivo si lo dan; no reutilizar "Walter".
- 2026-11-24: la API dice "Approved", pero falta confirmarlo en Business Support
  Home (punto 8 de la actualización).

### Prueba A — Sin portafolio nuevo (aísla H1)
- Cuenta de Facebook antigua del dueño, 2FA activada, método de pago puesto y un
  portafolio **verificado que no sea Pengi Studio**: es el dueño de la app y Meta
  no deja conectarlo por Embedded Signup (error 3441128).
- Número de repuesto con WhatsApp Business usado de verdad más de 7 días
  (ideal: 30).
- Conectar por coexistence. No hacer nada más durante 24 h.
- Sobrevive → H1 gana peso. Cae → seguir con la prueba B.

### Prueba B — Sin coexistence (aísla H2)
- Mismo portafolio verificado. Número nuevo solo de API (sin app WhatsApp
  Business), alta normal con registro.
- 24 h sin actividad.
- Sobrevive y A cayó → el problema es coexistence. Cae → H3 o H4.

### Prueba C — Actividad paso a paso (aísla H4)
Sobre el número que haya sobrevivido, un paso por día:
1. "Actualizar estado" de plantillas (solo lectura).
2. "Crear plantillas faltantes" una vez.
3. Un mensaje de texto de respuesta dentro de la ventana de 24 h.
4. Un mensaje con plantilla aprobada.
El paso tras el que caiga señala la causa.

### Prueba D — Perfil del negocio (aísla H3)
- Revisar categoría, descripción y sitio web del portafolio y del perfil de
  WhatsApp. Si figura como salud/medicina, repetir la prueba A con un perfil
  completo y una descripción clara: "software de gestión de citas", sin venta
  de productos médicos ni telemedicina.

## Revisión del código contra las políticas (2026-11-16)

Fuentes: WhatsApp Business Messaging Policy y Meta Terms for WhatsApp Business
Platform (ambas con fecha 23 sep 2026). No se pudo abrir el texto de "Terms of
Acceptable Use" ni la Commerce Policy por separado, así que la revisión no es
completa.

| Regla | Estado | Dónde |
|---|---|---|
| Solo escribir a quien dio consentimiento | Cumple en recordatorios, no-show, lista de espera, avisos del portal y plantillas del inbox | `sender.go:343`, `no-show.go:74,236`, `offer.go:98,160`, `notices.go:130`, `template-handler.go:142` |
| Respetar STOP / BAJA | Cumple: borra el consentimiento y no lo vuelve a pedir | `webhook-handler.go:636`, `consent-flow.go:47` |
| Fuera de 24 h solo plantillas | Cumple | `conversation-handler.go:168`, `bot-reply.go:200` |
| Solo plantillas aprobadas | Cumple desde #135 | `sender.go` (`sendTemplate`) |
| Bot con paso a una persona | Cumple (handoff a recepción, emergencias) | `bot_guard.go`, `models/bot.go` |
| No pedir tarjetas ni cédulas por chat | Sin indicios en el bot | — |
| Código de acceso del portal | Sin comprobación de consentimiento: lo pide el propio paciente | `portal/services/delivery.go` |
| Envío de prueba | Sin comprobación de consentimiento (número que escribe el admin) | `whatsapp-handler.go` (`SendTest`) |
| Consentimiento masivo (hasta 1000 pacientes) | Depende de que la clínica lo tenga de verdad | `patient-handler.go:340` |
| Datos de chats a un proveedor de IA / no entrenar modelos | No verificable en el código (contrato del proveedor) | `features/ai/services/bot.go` |
| Telemedicina / información de salud por chat (Messaging Policy §3) | Prohibición **condicional**: solo si la normativa aplicable prohíbe enviar esos datos a sistemas sin salvaguardas reforzadas. Las plantillas no llevan datos clínicos (`pengi_resultados_listos` avisa sin incluir resultados); el texto libre depende de la clínica. Falta criterio legal sobre la LOPDP de Ecuador | `templates.go:83` |
| Productos médicos y medicamentos (§4) | Prohíbe vender o promocionar **productos**; no menciona servicios (citas, recordatorios). Gentoo no vende productos por WhatsApp | — |
| "Verticales reguladas" en la app WhatsApp Business (§5) | Se refiere a juego, medicamentos sin receta y alcohol. No aplica a clínicas | — |
| Perfil e identidad del negocio exactos | Fuera del código: portafolio "Walter", cuenta "Pengi", app "Gentoo" | Meta |

## Caso de soporte a Meta (Tech Provider)
Abrir en Business Help Center con: App ID, ID de la WABA, los tres incidentes
con hora, que no hubo envíos masivos, y la pregunta concreta: qué actividad
incumplió los Terms of Acceptable Use y qué requisito de alta falta.
