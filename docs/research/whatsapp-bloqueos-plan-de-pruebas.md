# Bloqueos de WhatsApp al conectar: hechos y plan de pruebas

Estado al 2026-11-16. La causa de los bloqueos **no está confirmada**. Este
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
  historial, llamadas de plantillas). Sin evidencia.

## Reglas para toda prueba

- Solo números de repuesto. Nunca el número real de una clínica.
- Un solo intento de conexión por número. Sin desconectar ni reconectar.
- Producción con #134, #135 y #136 desplegados (nada crea plantillas solo).
- No pulsar "Crear plantillas faltantes" ni enviar mensajes hasta el paso
  indicado. Esperar 24 h entre pasos.
- Registrar en cada paso: hora, estado del número (WhatsApp Manager), estado
  del portafolio (Business Support Home) y logs de la API (`warn`/`error`).

## Pruebas

### Prueba 0 — Resultado de la revisión de "Walter"
- Restaurado → falso positivo automático; añadir método de pago y completar
  datos del negocio antes de cualquier otra prueba con ese portafolio.
- Rechazado → anotar el motivo si lo dan; no reutilizar "Walter".

### Prueba A — Sin portafolio nuevo (aísla H1)
- Cuenta de Facebook antigua del dueño, portafolio **Pengi Studio**
  (verificado), 2FA activada, método de pago puesto.
- Número de repuesto con WhatsApp Business usado de verdad más de 7 días.
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
