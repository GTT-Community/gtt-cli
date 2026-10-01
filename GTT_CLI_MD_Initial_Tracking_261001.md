# GTT CLI — Tracking de MD Iniciales Procesados

> **GTT Canonical Governance:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

## 1. Objetivo

Implementar en el CLI de GTT un mecanismo determinístico para registrar los archivos Markdown iniciales que el CLI ha leído, aplicado y validado.

> **Todo MD inicial procesado correctamente por el CLI debe quedar marcado como leído y aplicado.**

El developer no debe tener que registrar manualmente que una directiva ya fue procesada.

## 2. Responsabilidad

### CLI

El CLI es responsable de:
- detectar MD iniciales;
- leerlos;
- procesarlos;
- aplicar sus directivas;
- validar el resultado;
- registrar su estado;
- evitar reprocesamientos innecesarios;
- informar brevemente el resultado.

### Bootstrap

Bootstrap solo debe:
- crear/inicializar la estructura de tracking cuando el proyecto se inicializa;
- establecer los valores iniciales necesarios;
- no duplicar la lógica de procesamiento del CLI.

La lógica de estado y actualización pertenece al CLI.

## 3. Flujo obligatorio

```text
MD inicial
   ↓
DISCOVERED
   ↓
READ
   ↓
PROCESS
   ↓
APPLY
   ↓
VALIDATE
   ↓
MARK AS READ + APPLIED
```

El estado **READ + APPLIED** solo puede registrarse después de que la aplicación y validación hayan terminado correctamente.

## 4. Regla de fallo

Si el procesamiento o aplicación falla:

```text
MD
 ↓
READ
 ↓
PROCESS
 ↓
ERROR
 ↓
NO marcar como APPLIED
```

Nunca debe registrarse un MD como aplicado si la operación quedó incompleta, falló o no pudo validarse.

## 5. Estados mínimos

El mecanismo debe poder distinguir como mínimo:

```text
DISCOVERED
READ
APPLIED
VALIDATED
FAILED
```

La implementación puede utilizar un modelo equivalente si el CLI ya posee un sistema de estados compatible.

## 6. Reprocesamiento

Cuando el CLI encuentre un MD que ya está registrado como aplicado y validado:

```text
MD
 ↓
APPLIED + VALIDATED
 ↓
No reprocesar innecesariamente
 ↓
Continuar
```

Si el archivo cambió después de haber sido aplicado, el CLI debe detectar que el contenido/identidad relevante ya no coincide y tratarlo como un nuevo estado pendiente de procesamiento, según las reglas existentes de GTT.

No asumir que el nombre del archivo por sí solo identifica una versión.

## 7. Identidad y trazabilidad

El registro debe permitir determinar inequívocamente:
- qué MD fue procesado;
- qué versión/contenido fue procesado;
- cuándo fue procesado, si el sistema ya mantiene timestamps;
- resultado del procesamiento;
- si fue aplicado correctamente;
- si fue validado.

Preferir una identificación basada en contenido o mecanismo equivalente cuando sea compatible con la arquitectura existente.

## 8. Reporte al developer

El reporte normal debe ser extremadamente breve.

Ejemplo:

```text
✓ Directiva aplicada y validada: INITIAL-001.md
```

Si requiere acción humana:

```text
⚠ Directiva requiere autorización.
→ bash <script>.sh
```

> **Resumen mínimo por defecto; detalle bajo demanda.**

## 9. Integración con el flujo GTT

El tracking no debe convertirse en una nueva fuente paralela de verdad.

Debe utilizar los mecanismos de estado/índice existentes cuando corresponda.

Después de aplicar una directiva, el CLI debe validar el resultado antes de actualizar el estado.

## 10. Automatización y mínima fricción

> **El developer decide lo que solo el developer puede decidir. GTT hace todo lo demás.**

Por tanto:
- no preguntar si debe marcar el MD como leído;
- no preguntar si debe marcarlo como aplicado;
- no pedir confirmación para actualizar el tracking cuando la aplicación ya fue autorizada y completada;
- hacerlo automáticamente después de una validación exitosa.

La trazabilidad es una operación determinística.

## 11. Operaciones protegidas

Si aplicar el MD implica una operación protegida que requiere autorización humana:

```text
READ
 ↓
ANALYZE
 ↓
PREPARE
 ↓
GENERATE / USE .sh
 ↓
HUMAN AUTHORIZATION
 ↓
APPLY
 ↓
VALIDATE
 ↓
MARK READ + APPLIED
```

Nunca marcar antes de la autorización y aplicación efectiva.

## 12. Bootstrap

Si el proyecto se inicializa y todavía no existe la estructura necesaria para registrar estos estados, Bootstrap debe crearla.

Bootstrap no debe decidir por sí mismo que un MD fue aplicado.

**Bootstrap prepara la infraestructura. CLI mantiene el estado.**

## 13. Method Plans

La regla aplica a Light, Medium, Hard y Team.

La diferencia entre planes está en el nivel de autorización de determinadas operaciones, no en la obligación de mantener trazabilidad.

## 14. Trabajo para Claude

Implementa esta capacidad principalmente en el CLI de GTT.

Antes de modificar código:
1. inspecciona cómo el CLI descubre y procesa los MD iniciales;
2. identifica el mecanismo actual de estado/index/tracking;
3. identifica cómo Bootstrap inicializa la estructura;
4. reutiliza mecanismos existentes cuando sea posible;
5. no introduzcas un segundo sistema de verdad innecesario.

Después:
1. implementa el tracking de estados;
2. marca automáticamente READ/APPLIED después de aplicación exitosa;
3. valida antes de marcar;
4. evita reprocesamiento innecesario;
5. detecta cambios relevantes en el MD;
6. mantiene trazabilidad;
7. integra el comportamiento con los Method Plans;
8. mantiene reportes mínimos;
9. conserva las reglas actuales de autorización y governance.

### Criterio de aceptación

Debe ser posible ejecutar el flujo normal del CLI y obtener:

```text
✓ Directiva aplicada y validada: INITIAL-001.md
```

sin que el developer tenga que ejecutar un segundo comando para registrar que fue aplicada.

Si la aplicación falla:

```text
✗ Directiva no aplicada: INITIAL-001.md
```

y el estado **NO** debe quedar como APPLIED.

## 15. Validación

Antes de finalizar:
- ejecutar tests existentes;
- agregar tests para READ;
- agregar tests para APPLIED;
- agregar tests para VALIDATED;
- agregar test de fallo que confirme que no se marca APPLIED;
- agregar test de reprocesamiento;
- agregar test de MD modificado;
- verificar integración con Bootstrap;
- verificar que no se degrade el flujo actual;
- verificar que los reportes sean breves.

No crear commit.

## 16. Principio final

> **Una vez que GTT ha leído, aplicado y validado correctamente un MD inicial, debe saberlo y registrarlo automáticamente.**

> **El developer no debe administrar manualmente la memoria operativa del CLI.**
