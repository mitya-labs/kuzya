# Logo variants

Two colours on a dark tile: turquoise `#5EEADB` (light theme `#14968F`) and the contrast coral `#FF6B5A` (`#F2552C`). Palette and tile follow the Mitya Labs favicon.

| Files | Status | Idea |
|---|---|---|
| `kuzya.svg`, `kuzya-light.svg`, `kuzya.png` | earlier detailed version | Detailed domovoy: hat with a fold, eyebrows, cheeks, smile, hex nose, folded beard |
| `kuzya-simple.svg`, `kuzya-simple-light.svg`, `kuzya-simple.png` | rejected, too plain | Primitive: hat triangle, pentagon head, two dots, one smile line |
| `kuzya-mid.svg`, `kuzya-mid-light.svg`, `kuzya-mid.png` | **alternative** | Middle ground: `simple` plus the drooping hat tip, eye highlights, cheeks |
| `kuzya-mid-glasses.svg`, `kuzya-mid-glasses-light.svg`, `kuzya-mid-glasses.png` | **chosen** (used in the root `README.md`) | `mid` plus thin round glasses and two hair wisps under the hat |
| `kuzya-minimal.svg`, `kuzya-minimal-light.svg`, `kuzya-minimal.png` | rejected, too primitive | Claude-style minimalism: two triangles and two dots, no face details |

Русский: основной — `mid-glasses` (с очками), альтернативный — `mid` (без очков); `simple` и `minimal` отклонены, подробный `kuzya` оставлен как ранняя версия.
| `kuzya-terminal.png` | preview of the terminal logo | 34×38 pixel version of `mid-glasses` without the glasses (eyes with highlights) for the interactive mode (`cmd/kuzya/logo.go`, half-block characters, 24-bit colour or 256 colours); not shown when output is not a terminal, with `NO_COLOR` or `KUZYA_NOLOGO` |
