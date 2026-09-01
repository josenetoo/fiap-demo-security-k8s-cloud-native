# Segurança em Kubernetes com Cloud Native — Masterclass DOUGBR

Demo hands-on por **José Neto** (Cloud Solution Engineer & Tech Lead, Coordenador da PosTech de
DevOps e Arquitetura Cloud da FIAP), para o canal **DOUGBR - DevOps User Group Brazil**.

Jornada: `LAPTOP → CI/CD → CLUSTER`, cobrindo os 4C's. Roda local em `kind`
(base: Docker Desktop). O CI/CD roda ao vivo no GitHub Actions.

## 👉 Comece aqui: [HANDS-ON.md](HANDS-ON.md)

É o passo a passo único da live — setup, os 4 demos e o CI, com o comando, como
explicar e o resultado esperado de cada passo.

## Estrutura

```
HANDS-ON.md           o roteiro executável (comece por ele)
.github/workflows/    pipeline: security.yml (aprovado) + security-fail-demo.yml (reprovado)
demos/
  01-ataque/     pods de ataque
  02-build/      Dockerfiles + app (imagem insegura vs. distroless)
  03-admission/  políticas Kyverno + PSA + namespace dev + deploy seguro
  04-runtime/    regras Falco + pod-alvo
```

## Ferramentas

Trivy · Kyverno · Falco · Pod Security Admission · GitHub Actions. Referência: [Kubernetes — 4C's](https://kubernetes.io/pt-br/docs/concepts/security/overview/).
