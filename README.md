# Backend Jade's Açaí

API backend do projeto Jade's Açaí, desenvolvido em Go com Gin.

Consulte [ARCHITECTURE.md](ARCHITECTURE.md) para conhecer as camadas e as regras de dependência do backend.

## Requisitos

- Go 1.23 ou superior

## Executar localmente

```powershell
go run ./cmd/api
```

A API fica disponível em `http://localhost:8080` por padrão. Defina `PORT` para usar outra porta. Se houver um arquivo `.env` na raiz, suas variáveis são carregadas ao iniciar a API; variáveis já definidas no ambiente têm prioridade.

## Hot reload no desenvolvimento

Instale o Air com Go 1.25 ou superior:

```powershell
go install github.com/air-verse/air@latest
```

Na raiz do projeto, inicie o servidor com hot reload:

```powershell
air
```

O Air usa a configuração de [.air.toml](.air.toml), recompila a API quando arquivos Go mudam e reinicia o servidor. Se o comando `air` não for encontrado, adicione `$(go env GOPATH)\bin` ao `PATH`.

## Verificação de saúde

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Resposta esperada:

```json
{ "status": "ok" }
```

## Cardápio público

```http
GET /api/v1/menu/catalog
```

A API retorna itens e combos ativos do PostgreSQL, além das regras comerciais usadas pelo configurador. Itens ou combos pausados (`available: false`) ou arquivados não aparecem no menu público. `GET /api/v1/menu/combos` permanece disponível por compatibilidade e retorna somente os combos ativos.

```http
GET /api/v1/menu/combos
```

A resposta contém `{ "combos": [...] }`, com preços em centavos (`priceCents`) e IDs estáveis do catálogo.

`GET /api/v1/store/status` retorna `isOpen`, `manualOverride`, `timeZone` e as configurações públicas da loja. O calendário usa `America/Sao_Paulo`: terça a sexta das 19h às 23h, sábado e domingo das 17h às 23h; segunda-feira fica fechada por padrão.

## CORS

Por padrão, a API permite chamadas do frontend publicado em `https://jadesacai.vercel.app` e dos frontends locais em `http://localhost:3000` e `http://127.0.0.1:3000`. Para usar outra allowlist, configure `CORS_ALLOWED_ORIGINS` com a lista completa de origens permitidas, separadas por vírgulas, antes de iniciar a API. Essa configuração substitui a lista padrão. No Render, defina essa variável nas configurações do serviço (o arquivo `.env` local não é enviado ao deploy):

```powershell
$env:CORS_ALLOWED_ORIGINS = "https://jadesacai.vercel.app"
air
```

Somente os métodos `GET`, `POST` e `OPTIONS` são permitidos. As origens devem incluir protocolo e domínio, sem caminho.

## Segurança da API Admin

As rotas em `/api/v1/admin/*` exigem `Authorization: Bearer <ADMIN_API_KEY>`. Configure `ADMIN_API_KEY` no backend e no ambiente privado do frontend com o mesmo segredo aleatório de pelo menos 32 caracteres. No frontend, use somente uma variável server-side chamada `ADMIN_API_KEY`, nunca `NEXT_PUBLIC_ADMIN_API_KEY`. O Next valida a sessão `jade_admin_session` antes de encaminhar chamadas administrativas. A rota `GET /api/v1/admin/health` permite conferir a autenticação; sem segredo configurado ela responde `503`, e com uma credencial inválida responde `401`.

As configurações gerais usam `GET` e `PATCH /api/v1/admin/settings` para horários semanais, conteúdo da seção Nossa história e telefone WhatsApp. `PATCH /api/v1/admin/settings/override` recebe `{"manualOverride":true}`, `false` ou `null` para abrir, fechar ou voltar ao calendário automático.

## Recebimento de pedidos

```http
POST /api/v1/orders
Content-Type: application/json
```

O JSON contém `customer` (`name`, `phone`), `acai` com a primeira configuração para compatibilidade, `items` com as linhas do carrinho (`id`, `name`, `description`, `acai` e `estimatedSubtotalCents`), `delivery` (`postalCode`, `street`, `number`, `neighborhood`, `complement`, `reference`), `payment` (`method`: `pix`, `cash` ou `card`, `needsChange`, `changeForCents`), `notes` e `estimatedTotalCents` do pedido completo. `items` aceita até 100 configurações no mesmo pedido; cada linha representa um açaí montado, não um produto avulso.

`customer.phone` deve ser um celular brasileiro válido: DDD ativo e número com 9 dígitos iniciado por `9`. A API aceita o número formatado (`(21) 99999-9999`) ou somente dígitos e armazena a coluna de busca normalizada para dígitos.

A API grava cada pedido na tabela `orders` do PostgreSQL e responde `202 Accepted` com `{"status":"received","persisted":true,"orderId":"...","orderNumber":1,"orderDate":"2026-09-29"}`. Quando a loja está fechada pelo horário ou pelo controle manual, responde `409 Conflict` sem persistir. Em caso de outra falha ao gravar, responde `500` e não informa sucesso. Para pedidos com `items`, o backend recalcula `estimatedTotalCents` somando os subtotais informados e a taxa da zona; os preços dos itens ainda devem ser validados no backend antes de serem usados para cobrança. Pedidos legados sem `items` mantêm o total recebido por compatibilidade.

O checkout mostra a taxa da zona configurada para o bairro informado. O backend recalcula a taxa e o total a partir das linhas do pedido e rejeita endereços sem zona ativa com `422 Unprocessable Entity`. A configuração fica em `store_settings.deliveryZones`; por padrão, Realengo está habilitado por `300` centavos. Cada zona define `name`, `neighborhoods`, `feeCents` e `enabled`; os valores são editados em Configurações gerais no painel Admin. A origem de referência é Rua Nepomuceno, 12, Realengo, Rio de Janeiro. As taxas das demais zonas são manuais, sem cálculo automático por quilômetro.

O `orderId` aleatório permite acompanhar o pedido sem cadastro. O link público do frontend usa esse ID como credencial e consulta:

```http
GET /api/v1/orders/{orderId}/tracking
```

A resposta contém somente `orderNumber`, `orderDate`, `status` e `createdAt`; dados pessoais e endereço não são expostos. A rota responde `404` quando o pedido não existe e envia `Cache-Control: no-store`. Trate o link como privado e não o publique.

## Listagem Admin de pedidos

```http
GET /api/v1/admin/orders?date=2026-09-29&status=received&search=Ana&page=1&limit=20
Authorization: Bearer <ADMIN_API_KEY>
```

Todos os parâmetros são opcionais. A resposta inclui os pedidos mais recentes, os dados enviados no pedido (`orderData`), o dia comercial (`orderDate`), o número sequencial do dia (`orderNumber`) e paginação (`page`, `limit`, `total`). `limit` aceita de 1 a 100; `search` procura por nome ou telefone. O número reinicia em `1` a cada dia em `America/Sao_Paulo`; o UUID continua como identificador interno. A rota usa a autenticação Admin descrita acima.

Para atualizar uma etapa, use `PATCH /api/v1/admin/orders/{orderId}` com JSON como `{"status":"preparing"}`. Os status aceitos são `received`, `preparing`, `ready`, `out_for_delivery`, `delivered` e `completed`.

## Dashboard Admin

```http
GET /api/v1/admin/dashboard?date=2026-09-29
Authorization: Bearer <ADMIN_API_KEY>
```

`date` é opcional; por padrão a API usa o dia atual em `America/Sao_Paulo`. A resposta contém os totais do dia, contagens por status (`statusCounts`) e até cinco pedidos recentes (`recentOrders`). A rota usa a autenticação Admin descrita acima.

## Modelo do catálogo

A migration [20260929170000_create_catalog.sql](supabase/migrations/20260929170000_create_catalog.sql) cria `catalog_items`, `catalog_combos`, `catalog_combo_items` e `catalog_rules`, semeando itens e Combos. A migration [20261003120000_create_catalog_images_bucket.sql](supabase/migrations/20261003120000_create_catalog_images_bucket.sql) configura o bucket público `catalog-images` no Supabase Storage. A migration [20261005160000_add_store_settings.sql](supabase/migrations/20261005160000_add_store_settings.sql) semeia as configurações da loja. As migrations `20261006120000` a `20261006150000` foram etapas de transição do modelo Gourmet. A migration [20261006160000_split_gourmets_from_combos.sql](supabase/migrations/20261006160000_split_gourmets_from_combos.sql) cria `catalog_gourmet`, `catalog_gourmet_items` e `catalog_gourmet_sizes`, move os Gourmets existentes preservando IDs, receitas e opções de tamanho/preço, e remove a classificação e variantes antigas de Combo. `available = false` pausa um registro; `deleted_at` arquiva sem apagar referências. O menu público lê Combos e Gourmets em coleções separadas por `/api/v1/menu/catalog`.

### Rotas administrativas do catálogo

Todas exigem `Authorization: Bearer <ADMIN_API_KEY>`:

- `GET /api/v1/admin/catalog`: itens, Combos, Gourmets, receitas, variantes de tamanho/preço e regras comerciais.
- `POST /api/v1/admin/catalog/items`: cria flavor, size, topping, sauce, condiment position, fruit ou extra.
- `PATCH /api/v1/admin/catalog/items/{itemId}`: altera nome, preço, ordem ou disponibilidade.
- `DELETE /api/v1/admin/catalog/items/{itemId}`: arquiva o item sem apagar referências.
- `POST /api/v1/admin/catalog/combos`: cria um Combo personalizável vinculando tamanhos e itens existentes; `items` aceita várias porções com `quantity`, com soma total limitada a 100.
- `PATCH /api/v1/admin/catalog/combos/{comboId}`: altera dados ou substitui tamanhos e itens vinculados ao Combo.
- `DELETE /api/v1/admin/catalog/combos/{comboId}`: arquiva o combo.
- `POST /api/v1/admin/catalog/gourmets`: cria um Gourmet independente com `description`, receita fixa em `items` e opções `sizes` (`sizeItemId`, `priceCents`).
- `PATCH /api/v1/admin/catalog/gourmets/{gourmetId}`: altera produto, descrição, receita, tamanhos/preços, imagem, ordem ou disponibilidade.
- `DELETE /api/v1/admin/catalog/gourmets/{gourmetId}`: arquiva o Gourmet sem apagar o histórico de pedidos.

### Configurar Supabase

1. No Supabase, execute as migrations com `npx supabase db push --db-url "$env:DATABASE_URL"` (PowerShell) ou `npx supabase db push --db-url "$DATABASE_URL"` (bash). Elas criam pedidos, numeração diária e o catálogo inicial; as migrations estão em `supabase/migrations/`.
2. Copie a URI de conexão do projeto e substitua `[YOUR-PASSWORD]` pela senha do banco. Adicione `?sslmode=require` ao final da URI. Se a senha tiver caracteres especiais, use o formato URL-encoded.
3. Defina `DATABASE_URL` no `.env` local (use [.env.example](.env.example) como referência) e nas Environment Variables do serviço no Render. Nunca coloque essa URI no frontend/Vercel ou no Git.
4. Reinicie a API local ou faça um novo deploy no Render. A API valida a conexão ao iniciar e não sobe se `DATABASE_URL` estiver ausente ou inválida.

Se o Render não conseguir conectar ao endereço direto do banco, use a URI **Session pooler** exibida em **Supabase → Connect**. Não use uma chave `service_role` no frontend.

## Testes

```powershell
go test ./...
```
