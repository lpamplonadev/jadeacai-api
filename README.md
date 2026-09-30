# Backend Jade Açaí

API backend do projeto Jade Açaí, desenvolvido em Go com Gin.

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

## Catálogo de combos

```http
GET /api/v1/menu/combos
```

A resposta contém os combos atuais, suas inclusões e preços em centavos (`priceCents`). Esses dados refletem o catálogo do frontend no momento da implementação e ainda precisam de confirmação comercial; o frontend ainda não consome este endpoint.

## CORS

Por padrão, a API permite chamadas do frontend publicado em `https://jadeacai-web.vercel.app` e dos frontends locais em `http://localhost:3000` e `http://127.0.0.1:3000`. Para usar outra allowlist, configure `CORS_ALLOWED_ORIGINS` com a lista completa de origens permitidas, separadas por vírgulas, antes de iniciar a API. Essa configuração substitui a lista padrão. No Render, defina essa variável nas configurações do serviço (o arquivo `.env` local não é enviado ao deploy):

```powershell
$env:CORS_ALLOWED_ORIGINS = "https://jadeacai-web.vercel.app"
air
```

Somente os métodos `GET`, `POST` e `OPTIONS` são permitidos. As origens devem incluir protocolo e domínio, sem caminho.

## Segurança da API Admin

As rotas em `/api/v1/admin/*` exigem `Authorization: Bearer <ADMIN_API_KEY>`. Configure `ADMIN_API_KEY` no backend e no ambiente privado do frontend com o mesmo segredo aleatório de pelo menos 32 caracteres. No frontend, use somente uma variável server-side chamada `ADMIN_API_KEY`, nunca `NEXT_PUBLIC_ADMIN_API_KEY`. O Next valida a sessão `jade_admin_session` antes de encaminhar chamadas administrativas. A rota `GET /api/v1/admin/health` permite conferir a autenticação; sem segredo configurado ela responde `503`, e com uma credencial inválida responde `401`.

## Recebimento de pedidos

```http
POST /api/v1/orders
Content-Type: application/json
```

O JSON deve conter `customer` (`name`, `phone`), `acai` (`flavorId`, `sizeId`, `comboId` opcional, `toppingIds`, `sauceId`, `condimentPositionId`, `fruitIds`, `extraIds`), `delivery` (`postalCode`, `street`, `number`, `neighborhood`, `complement`, `reference`), `payment` (`method`: `pix`, `cash` ou `card`, `needsChange`, `changeForCents`), `notes` e `estimatedTotalCents`.

A API grava cada pedido na tabela `orders` do PostgreSQL e responde `202 Accepted` com `{"status":"received","persisted":true,"orderId":"...","orderNumber":1,"orderDate":"2026-09-29"}`. Em caso de falha ao gravar, responde `500` e não informa sucesso. `estimatedTotalCents` ainda é informado pelo cliente e não é recalculado pela API; valide os preços no backend antes de usar esse valor para cobrança.

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

A migration [20260929170000_create_catalog.sql](supabase/migrations/20260929170000_create_catalog.sql) cria `catalog_items`, `catalog_combos`, `catalog_combo_items` e `catalog_rules`, semeando os itens e combos atuais do frontend. `available = false` pausa um registro sem removê-lo da loja; `deleted_at` permite arquivá-lo sem quebrar combos ou o histórico dos pedidos. Combos referenciam tamanhos e itens por chaves estrangeiras. As rotas protegidas de CRUD estão descritas abaixo; a próxima etapa conecta a tela Admin e o menu público.

### Rotas administrativas do catálogo

Todas exigem `Authorization: Bearer <ADMIN_API_KEY>`:

- `GET /api/v1/admin/catalog`: itens, combos, componentes de cada combo e regras comerciais.
- `POST /api/v1/admin/catalog/items`: cria flavor, size, topping, sauce, condiment position, fruit ou extra.
- `PATCH /api/v1/admin/catalog/items/{itemId}`: altera nome, preço, ordem ou disponibilidade.
- `DELETE /api/v1/admin/catalog/items/{itemId}`: arquiva o item sem apagar referências.
- `POST /api/v1/admin/catalog/combos`: cria combo vinculando IDs de tamanho e itens existentes.
- `PATCH /api/v1/admin/catalog/combos/{comboId}`: altera dados ou substitui os itens vinculados.
- `DELETE /api/v1/admin/catalog/combos/{comboId}`: arquiva o combo.

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
