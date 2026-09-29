# Backend Jade Açaí

API backend do projeto Jade Açaí, desenvolvido em Go com Gin.

## Requisitos

- Go 1.23 ou superior

## Executar localmente

```powershell
go run ./cmd/api
```

A API fica disponível em `http://localhost:8080` por padrão. Defina `PORT` para usar outra porta.

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

## Recebimento de pedidos

```http
POST /api/v1/orders
Content-Type: application/json
```

O JSON deve conter `customer` (`name`, `phone`), `acai` (`flavorId`, `sizeId`, `comboId` opcional, `toppingIds`, `sauceId`, `condimentPositionId`, `fruitIds`, `extraIds`), `delivery` (`postalCode`, `street`, `number`, `neighborhood`, `complement`, `reference`), `payment` (`method`: `pix`, `cash` ou `card`, `needsChange`, `changeForCents`), `notes` e `estimatedTotalCents`.

A API valida os campos obrigatórios e responde `202 Accepted` com `{"status":"received","persisted":false}`. Por enquanto, o pedido não é armazenado nem encaminhado ao painel; `estimatedTotalCents` é informado pelo cliente e ainda não é recalculado pela API. O frontend também ainda não está integrado a esta rota.

## Testes

```powershell
go test ./...
```
