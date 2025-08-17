# go-loadbalancer
Um Load Balancer  TCP em Golang

**LB TCP com foco em performance** escrito em Go, suportando:

- Conexões **persistentes** com pool
- **Sticky sessions** (por IP)
- **Round-robin load balancing**
- **Backpressure** e gerenciamento de erros bidirecional
- **PROXY Protocol v1** opcional
- Health check ativo e passivo
- Suporte a múltiplos processos via **SO_REUSEPORT**

---

## Componentes

### Listener
- Aceita conexões TCP de clientes.
- Suporta múltiplos processos para escalabilidade (`SO_REUSEPORT`).
- Configurável via `LISTEN` (porta/endereço).

### Load Balancer
- Distribui conexões para backends vivos.
- **Modos**:
  - `rr` → Round-robin
  - `sticky` → Consistent hash por IP do cliente
- Permite sticky sessions e fallback caso backend esteja offline.

### Backend
- Representa um servidor de destino TCP.
- Cada backend tem:
  - Flag `Alive`
  - Pool de conexões persistentes
  - Peso para balanceamento futuro (expandível)

### Connection Pool
- Mantém **conexões persistentes** com o backend.
- Configurações:
  - `PoolSizePerBackend` → máximo de conexões persistentes
  - `DialTimeout` e `IdleTimeout`
- Evita latência de `connect` repetido.
- Limpa conexões antigas automaticamente.
- Suporta burst de conexões extras sem quebrar o pool.

### Pump (Pipeline bidirecional)
- Faz cópia de dados **cliente ↔ backend**.

### Backpressure & Erro Handling
- Marca conexões com problemas como “mortas” e remove do pool.
- Garante que **erros não travem o fluxo de dados**.

### Sticky Sessions
- Implementado por hash do IP do cliente.
- Pode ser adaptado para tokens ou IDs de sessão se houver protocolo L7.

### Health Check
- Loop ativo e passivo:
  - Conexões periódicas rápidas para checar backend
  - Limpeza de conexões antigas no pool


# Como rodar / testar

### Backends de teste
go run tools/test_backend.go 5001 &
go run tools/test_backend.go 5002 &


### LB
export LISTEN=":4000"
export BACKENDS="127.0.0.1:5001,127.0.0.1:5002"
export LB_MODE="rr"
export POOL_SIZE=256
make run


### Conecta no LB
nc 127.0.0.1 4000 # keepAlive connection

echo "a" | python3 tools/send.py 127.0.0.1 4000
