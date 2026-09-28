# Los mensajes de respuesta son strings; el test de claves es el candado, no el tipo

El `Message` de `envelope.Response` es un `string` que debe ser una clave del catálogo de mensajes (ADR 0003). Toda respuesta, también la de un middleware o la de un handler que descarga archivos, se escribe con `envelope.Handle`, `envelope.Write` o `envelope.Abort`, que traducen el mensaje y el código de error al idioma del pedido; nadie llama a `c.JSON` con un envelope.

Evaluamos tipar el mensaje (`type MessageKey string`) para que el compilador rechace frases donde va una clave, y lo descartamos: en Go una constante string sin tipo se convierte sola a un tipo con nombre, así que `envelope.ErrorResponse(400, "Company not found", …)` seguiría compilando. Todas las llamadas pasan literales, por lo que el tipo no atraparía nada que no atrape ya el test de claves del catálogo (`TestEmbedded_EnvelopeMessagesAreCatalogKeys`), que recorre el AST y exige que cada literal exista en ambos idiomas.

## Considered Options

- **`type MessageKey string`**: descartado. No cambia ninguna llamada y tampoco protege nada (conversión implícita de constantes).
- **Constantes generadas desde `messages_es.json`** (`msg.CompanyNotFound`) o un tipo opaco que solo un paquete generado puede construir: descartado por ahora. Da seguridad en compilación, pero exige editar las ~730 llamadas, agregar un paso de generación de código al build y choca con el campo `MessageKey` que ya usan las notificaciones; el test de claves da la misma garantía en CI.

## Consequences

Una respuesta que no pase por `Handle`/`Write`/`Abort` sale sin traducir, y el test de claves no lo detecta: revisar que middleware y descargas usen `envelope.Write`/`envelope.Abort`. Si algún día los mensajes se arman en runtime (variables, `fmt.Sprintf`), el test AST deja de cubrirlos y conviene reabrir la opción de constantes generadas.
