# El aislamiento por tenant lo aplica un plugin de GORM, no cada handler

Hasta ahora cada handler debía acordarse de filtrar por tenant (`Scopes(TenantScope(c))`). En unos 90 lugares varios lo olvidaron o usaron `AuditScope`, que no filtra. Eso permitió leer y modificar citas, signos vitales e historias clínicas de otras empresas (fixes `4cd158e` y `f2e1973`).

Decidimos que un plugin de GORM aplique el aislamiento. `TenantMiddleware` deja el tenant en el contexto de la petición. El plugin agrega `tenant_id = ?` a toda consulta, actualización y borrado sobre modelos con `TenantID`, y rellena `TenantID` al crear. Sin tenant en el contexto, la operación falla en vez de devolver datos de todos. Los workers y schedulers, que trabajan sobre todos los tenants, usan una salida explícita y fácil de buscar (`tenantdb.System`).

Las tablas sin `tenant_id` (signos vitales, SOAP, recetas, items de comprobantes) heredan el tenant de su padre. Solo se accede a ellas a través de un padre ya filtrado, o después de verificar que el padre pertenece al tenant.

## Considered Options

- **Helper explícito por handler** (`tenantdb.Scoped(c)`): solo le cambia el nombre a lo que ya existía. Olvidarlo sigue siendo posible.
- **Repositorios por entidad**: el aislamiento quedaría igual de garantizado, pero habría que reescribir todos los handlers.
- **Agregar `tenant_id` a las tablas hijas**: descartado por ahora. Implica migraciones y backfill en muchas tablas, y los accesos directos por ID a tablas hijas son pocos y están verificados.

## Consequences

La migración va por etapas: primero el plugin, activo solo cuando hay tenant en el contexto; después una feature por commit. Cuando todo esté migrado, el plugin pasa a fallar sin tenant, y un test en CI impide volver a escribir consultas que se salten el módulo.
