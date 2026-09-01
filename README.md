# Segurança em Kubernetes com Cloud Native

Laboratório de segurança cloud native em Kubernetes, executado localmente em `kind` sobre Docker Desktop. A jornada `LAPTOP → CI/CD → CLUSTER` cobre os 4Cs.

## Roteiro operacional: [HANDS-ON.md](HANDS-ON.md)

Contém o setup, quatro cenários e o CI, com comandos e resultados esperados.

## Estrutura

```
HANDS-ON.md           roteiro operacional
.github/workflows/    gates de configuração/segredos e análise informativa da imagem
demos/
  01-ataque/     pods de ataque
  02-build/      Dockerfiles + app (imagem insegura vs. distroless)
  03-admission/  políticas Kyverno + PSA + namespace dev + deploy seguro
  04-runtime/    regras Falco + pod-alvo
```

## Ferramentas

Trivy · Kyverno · Falco · Pod Security Admission · GitHub Actions. Referência: [Kubernetes — 4C's](https://kubernetes.io/pt-br/docs/concepts/security/overview/).
