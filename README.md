# Exemplo robusto de Saga Pattern (Event-Driven)

Este projeto mostra um **microserviço com Saga Pattern orientado a eventos**, simulando um fluxo de e-commerce com bastante observabilidade.

## Cenário

Fluxo principal da saga (`OrderSaga`):

1. Reservar estoque
2. Cobrar pagamento
3. Criar envio
4. Confirmar pedido

Quando há falha, a orquestração executa compensações:

- Falha no pagamento -> libera estoque
- Falha no envio -> estorna pagamento e libera estoque
- Falha no estoque -> encerra sem compensação (nada foi comprometido antes)

## Estrutura

- `saga/event_bus.py`: barramento de eventos com histórico.
- `saga/services.py`: serviços de Estoque, Pagamento e Envio.
- `saga/orchestrator.py`: orquestrador da saga e lógica de compensação.
- `demo.py`: execução manual com logs detalhados.
- `tests/test_saga.py`: cenários de sucesso e falha.

## Como rodar

```bash
python demo.py
```

## Como testar

```bash
python -m unittest discover -s tests -v
```

## Logs

O sistema gera logs detalhados para:

- publicação e consumo de eventos,
- transição de estado da saga,
- falhas de cada etapa,
- compensações executadas.

Isso ajuda a visualizar claramente o comportamento em sucesso e falha.
