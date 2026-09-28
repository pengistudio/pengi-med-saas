# Los textos viven en el catálogo de mensajes embebido en el binario, no en una tabla

Todos los textos de la API y de las apps web (mensajes de respuesta, códigos de error, etiquetas) están en `apps/api/i18n/messages/messages_es.json` y `messages_en.json`, embebidos en el binario. Hasta ahora, en cada arranque se copiaban a la tabla `messages` (una consulta por clave), un caché global los leía de esa tabla y había un endpoint sin autenticación para recargarlos. La tabla nunca fue la fuente de verdad: cualquier cambio hecho en ella se perdía en el siguiente arranque.

Decidimos que el catálogo de mensajes (`i18n/catalog`) se cargue directamente del JSON embebido al arrancar y se pase como dependencia al middleware de i18n y al handler de mensajes. No hay tabla, ni caché global, ni recarga en caliente: cambiar un texto requiere un deploy. `GET /api/v1/i18n/messages` devuelve un mapa plano `{clave: valor}` del idioma pedido y un `ETag` con el hash del contenido, para que el navegador solo descargue los textos cuando cambian (304 si no cambiaron).

Como los textos van con el código, se validan con tests en CI: es y en tienen las mismas claves y los mismos placeholders, todo código de error de `core/errors/codes.go` tiene traducción, y todo mensaje literal que un handler pasa a `envelope` existe como clave en ambos idiomas. Las frases heredadas que aún no son claves están en `i18n/catalog/testdata/legacy_messages.txt`; el test falla si aparece una nueva y si una de la lista ya no existe, así que la lista solo puede achicarse.

## Considered Options

- **Mantener la tabla como fuente de verdad** (editar textos sin deploy): descartado. Hoy nadie edita textos fuera del repositorio, y la tabla se sobrescribía en cada arranque, así que solo sumaba consultas al arranque, un endpoint de recarga sin autenticación y la posibilidad de que los datos diverjan del código.
- **Caché global con recarga en caliente**: descartado. Un estado global dificulta los tests, y sin tabla no hay nada que recargar.

## Consequences

Si algún día hace falta editar textos desde el backoffice, se implementa como una capa de overrides por encima del catálogo (el catálogo embebido sigue siendo la base y los tests siguen validándolo), no volviendo a una tabla que reemplace al JSON. La migración `DB20260927_6` borra la tabla `messages`.
