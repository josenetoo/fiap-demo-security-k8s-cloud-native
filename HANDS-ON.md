# 🎬 Masterclass — Segurança em Kubernetes com Cloud Native

**Evento:** DOUGBR — DevOps User Group Brazil
**Instrutor:** José Neto — Cloud Solution Engineer & Tech Lead · Coordenador da PosTech de DevOps e Arquitetura Cloud da FIAP
**Temas:** Os 4C's · Ataque a um cluster · Trivy + CI/CD · Admissão (PSA + Kyverno) · Runtime (Falco)
**Formato:** quatro demonstrações ao vivo, na jornada `LAPTOP → CI/CD → CLUSTER`

> Este é o roteiro para seguir passo a passo durante a sessão. Cada passo traz
> **o que dizer**, **o comando** e **o resultado esperado**. Os comandos podem
> ser copiados um a um, na ordem.

---

## 🚀 Antes de Começar

> Esta masterclass exige um cluster Kubernetes em execução. A **Parte 0** reúne
> todo o setup — pode ser executado com antecedência ou ao vivo, deixando as
> instalações rodando enquanto se apresenta o conteúdo dos 4C's.

### Base: Docker Desktop

Partimos do **Docker Desktop**, que já entrega **Docker** e **kubectl** prontos
e ainda provisiona um **cluster Kubernetes de três nós** (baseado em kind).
É o caminho mais direto para os participantes acompanharem.

1. Instale e abra o Docker Desktop.
2. **Settings › Resources › Memory:** defina **8 GB** (Kyverno e Falco exigem memória).
3. **Settings › Kubernetes › Enable Kubernetes › Apply & Restart.** Aguarde o indicador ficar verde (1–2 min).

Confirme o Docker:
```bash
docker version
```

Confirme o cluster (contexto e nós):
```bash
kubectl config current-context
```
```bash
kubectl get nodes
```

**Resultado esperado:** contexto `docker-desktop` e três nós `Ready`
(`desktop-control-plane`, `desktop-worker`, `desktop-worker2`).

> O cluster do Docker Desktop chama-se `desktop` — esse é o nome usado pelo
> `kind load`. Não é necessário criar cluster pela linha de comando.

### Instalar as ferramentas restantes

Faltam o `kind` (usado para carregar imagens no cluster), o `helm` e as
ferramentas de segurança. Verifique o que está ausente:

```bash
for t in kind helm trivy jq gh; do command -v "$t" >/dev/null && echo "ok   $t" || echo "faltando $t"; done
```

**macOS (Homebrew):**
```bash
brew install kind helm trivy jq gh
```

**Linux (Homebrew) — ou use os binários oficiais de cada projeto:**
```bash
brew install kind helm trivy jq gh
```

**Windows (PowerShell / winget):**
```powershell
winget install Kubernetes.kind Helm.Helm AquaSecurity.Trivy jqlang.jq GitHub.cli
```

Autentique a GitHub CLI (necessária no bloco 2):
```bash
gh auth login
```

> Os comandos das demonstrações (`kubectl`, `trivy`, `helm`, `kind`) são
> idênticos em macOS, Linux e Windows. A única diferença entre sistemas está na
> instalação das ferramentas, acima.

---

## ⚙️ Parte 0: Setup

> As imagens permanecem locais: o Trivy as analisa diretamente do Docker e os
> pods as recebem via `kind load`. Não é necessário registry nem Docker Hub.
> Os passos 0.1 e 0.2 levam alguns minutos — inicie-os e prossiga com a
> explicação dos 4C's enquanto instalam.

### Passo 0.1: Instalar Kyverno e Falco

Adicione o repositório do Kyverno:
```bash
helm repo add kyverno https://kyverno.github.io/kyverno/ && helm repo update
```

Instale o Kyverno:
```bash
helm upgrade --install kyverno kyverno/kyverno --namespace kyverno --create-namespace --wait --timeout 5m
```

Adicione o repositório do Falco:
```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts && helm repo update
```

Instale o Falco com as regras da sessão:
```bash
helm upgrade --install falco falcosecurity/falco --namespace falco --create-namespace --set driver.kind=modern_ebpf --set tty=true --set falco.json_output=false --set-file 'customRules.dougbr-rules\.yaml'=demos/04-runtime/regras-dougbr.yaml --wait --timeout 5m
```

> Se o Falco falhar com `modern_ebpf` (kernel sem BTF), repita o comando acima
> trocando por `--set driver.kind=ebpf`.

### Passo 0.2: Construir as imagens e carregar no cluster

Imagem insegura (base completa, executa como root):
```bash
docker build -f demos/02-build/Dockerfile.inseguro -t dougbr/app:inseguro demos/02-build
```

Imagem segura (multi-stage, distroless, nonroot):
```bash
docker build -f demos/02-build/Dockerfile.seguro -t dougbr/app:seguro demos/02-build
```

Carregue as duas no cluster — é assim que o pod recebe a imagem, sem registry:
```bash
kind load docker-image dougbr/app:seguro --name desktop
```
```bash
kind load docker-image dougbr/app:inseguro --name desktop
```

### Passo 0.3: Publicar o repositório no GitHub (para o pipeline do bloco 2)

Confirme a autenticação:
```bash
gh auth status || gh auth login
```

Se o repositório **ainda não existe**, crie e publique (ajuste o nome se desejar):
```bash
gh repo create dougbr-security --public --source=. --remote=origin --push
```

Se o repositório **já existe** (foi criado antes das últimas alterações),
publique o estado atual — incluindo os workflows atualizados:
```bash
git add -A
```
```bash
git commit -m "Ajustes na masterclass DOUGBR"
```
```bash
git push
```

> O push para a branch `main` dispara automaticamente o workflow
> `security-pipeline` (a execução aprovada). Ela será acompanhada ao vivo no Passo 6.

> Se o `git commit` indicar identidade não configurada, defina-a uma vez:
> `git config user.email "seu@email"` e `git config user.name "Seu Nome"`.

### ✅ Confirmar o ambiente

```bash
kubectl get nodes
```
```bash
kubectl -n kyverno get pods
```
```bash
kubectl -n falco get pods
```
```bash
docker images | grep dougbr/app
```

Esperado: três nós `Ready`, pods `Running` no Kyverno e no Falco, e as duas
imagens `dougbr/app` (`seguro` e `inseguro`) presentes no Docker.

---

## 📚 Parte 1 — Demo 1: O Ataque

> **Objetivo:** demonstrar que, sem controles de admissão, um simples
> `kubectl apply` concede acesso de `cluster-admin`. Dois vetores, nenhum exploit.

### Passo 1: Container escape via pod privilegiado

**O que dizer:**
> "Vou aplicar um pod com a mesma configuração que muitos agentes de
> monitoramento pedem: privilegiado, com o disco do nó montado internamente.
> Não há exploit — é apenas a especificação do pod, como a API permite."

Crie o namespace:
```bash
kubectl apply -f demos/01-ataque/00-namespace.yaml
```

Aplique o pod privilegiado no namespace `ataque`:
```bash
kubectl -n ataque apply -f demos/01-ataque/10-pod-privilegiado.yaml
```

Aguarde ficar pronto:
```bash
kubectl -n ataque wait --for=condition=Ready pod/pod-privilegiado --timeout=60s
```

Acesse o pod:
```bash
kubectl -n ataque exec -it pod-privilegiado -- sh
```

Dentro do pod — o disco do nó está montado em `/host`:
```sh
ls -la /host/etc/kubernetes/
```
```sh
cat /host/etc/kubernetes/admin.conf | head -20
```
```sh
ls -la /host/etc/kubernetes/pki/
```
```sh
exit
```

**Resultado esperado:** o arquivo `admin.conf` (kubeconfig de administrador do
cluster) e o diretório `pki/` com a autoridade certificadora ficam acessíveis.

> "Este `admin.conf` é a chave-mestra do cluster. Copiado para outra máquina,
> concede acesso de `cluster-admin` a qualquer momento. Um único `privileged:
> true` colado de um tutorial comprometeu o cluster inteiro."

### Passo 2: O pod sem indicadores de risco aparentes

**O que dizer:**
> "Este segundo pod não tem `privileged`, `hostPath` nem `hostNetwork`.
> Passaria em qualquer revisão de código. O único problema é a ServiceAccount
> que ele utiliza."

Crie a ServiceAccount e o ClusterRoleBinding permissivo:
```bash
kubectl apply -f demos/01-ataque/20-rbac-generoso.yaml
```

Aplique o pod comum:
```bash
kubectl apply -f demos/01-ataque/30-pod-comum.yaml
```

Aguarde ficar pronto:
```bash
kubectl -n ataque wait --for=condition=Ready pod/pod-comum --timeout=60s
```

Acesse o pod:
```bash
kubectl -n ataque exec -it pod-comum -- sh
```

Dentro do pod — o token da ServiceAccount lê todos os secrets do cluster:
```sh
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
```
```sh
apk add --no-cache curl >/dev/null 2>&1
```
```sh
curl -sk -H "Authorization: Bearer $TOKEN" https://kubernetes.default.svc/api/v1/secrets | head -30
```
```sh
exit
```

**Resultado esperado:** a API retorna a lista de secrets do cluster.

> "Dois caminhos diferentes, o mesmo desfecho. Nenhum campo indicava perigo.
> Revisar manifestos manualmente não escala — precisamos de controles
> automáticos. É exatamente para onde vamos."

### 🧹 Limpeza

```bash
kubectl delete ns ataque --ignore-not-found
```
```bash
kubectl delete clusterrolebinding app-legado-cluster-admin --ignore-not-found
```

---

## 🔍 Parte 2 — Demo 2: Build + CI/CD

> **Objetivo:** deslocar a segurança para antes do cluster — primeiro na estação
> de trabalho (shift-left), depois de forma obrigatória no CI (gate).

### Passo 3: Análise de vulnerabilidades — base completa vs. distroless

**O que dizer:**
> "A mesma aplicação em duas imagens: uma com base completa, outra distroless.
> Observe a diferença no número de vulnerabilidades."

Analise a imagem insegura:
```bash
trivy image --scanners vuln --severity HIGH,CRITICAL dougbr/app:inseguro
```

Analise a imagem segura:
```bash
trivy image --scanners vuln --severity HIGH,CRITICAL dougbr/app:seguro
```

**Resultado esperado** (os números variam conforme a data; o que importa é o contraste):
```
dougbr/app:inseguro   sistema operacional: dezenas de HIGH/CRITICAL  + binário: várias
dougbr/app:seguro     sistema operacional: 0                          + binário: 0 ou poucas
```

> "A diferença não está no código, e sim em tudo o que foi empacotado junto. O
> distroless zerou a superfície do sistema operacional. O que permanecer no
> binário vem da biblioteca padrão do Go, e recompilar com uma versão mais
> recente reduz esses itens. O ponto do distroless é reduzir o que fica
> disponível para o atacante: sem shell, boa parte dos ataques não se sustenta."

### Passo 4: Configuração incorreta e segredos

**O que dizer:**
> "O Trivy não analisa apenas imagens. Analisa Dockerfiles e manifestos, e
> detecta segredos."

Configuração incorreta no Dockerfile:
```bash
trivy config demos/02-build/Dockerfile.inseguro
```

Segredo embutido no código:
```bash
trivy fs --scanners secret demos/02-build/
```

**Resultado esperado:** o Trivy aponta a ausência de `USER` no Dockerfile e
detecta a credencial AWS embutida (`AWS_ACCESS_KEY_ID`).

> "Um segredo no Dockerfile vira camada de imagem, vira histórico. O Git não esquece."

### Passo 5: SBOM — o inventário da imagem

**O que dizer:**
> "O SBOM é o inventário de tudo o que existe na imagem. Quando surgir a próxima
> vulnerabilidade crítica de mercado, a pergunta será 'estou afetado?', e o
> SBOM permite responder rapidamente, sem levantamento manual."

Gere o SBOM:
```bash
trivy image --format cyclonedx --output /tmp/sbom.json dougbr/app:seguro
```

Conte os componentes catalogados:
```bash
jq '.components | length' /tmp/sbom.json
```

**Resultado esperado:** o número de componentes presentes na imagem.

### Passo 6: O pipeline de CI ao vivo

**O que dizer:**
> "Tudo o que fizemos na estação de trabalho, agora de forma obrigatória, no
> GitHub Actions."

**6a. A execução aprovada** (disparada pelo push do Passo 0.3).

Acompanhe pelo terminal:
```bash
gh run watch
```

Ou abra no navegador para exibir os jobs à audiência:
```bash
gh run view --web
```

**Resultado esperado:** os jobs `code-scan` e `build-scan` concluem com sucesso;
o SBOM é publicado como artefato.

> "A verificação local orienta; o gate do CI é obrigatório. Código que não passa
> no gate não avança para o merge."

**6b. A execução reprovada** (dispare ao vivo — conclui em cerca de 30 segundos).

Dispare o workflow que constrói a imagem insegura:
```bash
gh workflow run security-fail-demo.yml
```

Aguarde alguns segundos e acompanhe:
```bash
gh run watch
```

**Resultado esperado:** o job falha no gate, barrado pela credencial AWS e pela
configuração incorreta do Dockerfile.

> "O gate interrompe o build. Um relatório apenas informa; o gate impede.
> É essa a diferença entre observar e prevenir."

> **Rede:** caso o GitHub Actions demore ou fique indisponível ao vivo, utilize
> as capturas de tela feitas no ensaio. O ponto — aprovado vs. reprovado — não
> depende de aguardar a execução.

---

## 🚪 Parte 3 — Demo 3: Admissão

> **Objetivo:** o CI não impede um `kubectl apply` direto. O controle efetivo é
> a admissão. O mesmo pod do bloco 1 retorna — e desta vez não sobe.

### Passo 7: Pod Security Admission (nativo) — namespace `producao`

**O que dizer:**
> "Antes de instalar qualquer coisa: o Kubernetes já traz o Pod Security
> Admission. É apenas um label no namespace. Vou tentar aplicar o mesmo pod do
> bloco 1."

Crie o namespace `producao` com o perfil `restricted`:
```bash
kubectl apply -f demos/03-admission/policies/00-psa-labels.yaml
```

Tente aplicar o pod do ataque:
```bash
kubectl -n producao apply -f demos/01-ataque/10-pod-privilegiado.yaml
```

**Resultado esperado:** o pod é recusado pelo Pod Security com a mensagem
`violates PodSecurity "restricted:latest"`.

> "Se houver uma única prioridade ao sair daqui, é aplicar
> `pod-security.kubernetes.io/enforce: restricted` nos namespaces. Custo zero,
> e já teria bloqueado o ataque inicial."

### Passo 8: Kyverno — namespace `dev`

**O que dizer:**
> "O PSA é o perfil pronto; o Kyverno é a regra sob medida. Para deixar claro
> quem realiza o bloqueio, uso o namespace `dev`, que não tem PSA — aqui quem
> barra é o Kyverno, com a mensagem que definimos."

Crie o namespace `dev` (sem PSA):
```bash
kubectl apply -f demos/03-admission/policies/05-namespace-dev.yaml
```

Aplique a política que bloqueia contêineres privilegiados:
```bash
kubectl apply -f demos/03-admission/policies/10-bloquear-privilegiado.yaml
```

Aplique a política que bloqueia `hostPath`:
```bash
kubectl apply -f demos/03-admission/policies/20-bloquear-hostpath.yaml
```

Aplique a política que exige execução como não-root:
```bash
kubectl apply -f demos/03-admission/policies/30-exigir-nonroot.yaml
```

Confira as políticas:
```bash
kubectl get clusterpolicy
```

Tente aplicar o pod do ataque em `dev`:
```bash
kubectl -n dev apply -f demos/01-ataque/10-pod-privilegiado.yaml
```

**Resultado esperado:** o pod é bloqueado pelo Kyverno, com a mensagem em
português definida na política ("Container privilegiado nao e permitido...").

> Observação: o Kyverno 1.19 exibe um aviso de que o `ClusterPolicy` será
> substituído pelas novas APIs baseadas em CEL. O próprio Kyverno está migrando
> para o modelo CEL, alinhado ao `ValidatingAdmissionPolicy` nativo do Kubernetes.

### Passo 9: Mutação — o Kyverno também corrige

**O que dizer:**
> "Bloquear gera atrito com o time de desenvolvimento. O Kyverno também realiza
> mutação: injeta um securityContext seguro quando o autor esquece. O caminho
> seguro passa a ser o padrão."

Aplique a política de mutação:
```bash
kubectl apply -f demos/03-admission/policies/40-mutar-defaults-seguros.yaml
```

Crie um pod sem securityContext e observe o que o Kyverno injeta:
```bash
kubectl -n dev run teste --image=dougbr/app:seguro --dry-run=server -o yaml | grep -A6 securityContext
```

**Resultado esperado:** o `securityContext` seguro (runAsNonRoot, drop de
capabilities, seccomp) aparece injetado, sem ter sido informado no comando.

### Passo 10: O deploy que atende a todas as políticas

**O que dizer:**
> "E o deploy que respeita todas as regras? Sobe sem obstáculos."

Aplique o deploy seguro:
```bash
kubectl apply -f demos/03-admission/90-deploy-seguro.yaml
```

Acompanhe a subida (Ctrl+C ao estabilizar):
```bash
kubectl -n producao get pods -w
```

**Resultado esperado:** os pods `app` alcançam o estado `Running` (1/1),
utilizando a imagem carregada via `kind load`.

> "A admissão é o portão do cluster. Nada entra sem passar por ele — nem
> configuração incorreta, nem imagem sem procedência."

### Passo 11: Fechando o ciclo de supply chain (apenas explicação)

**O que dizer:**
> "A etapa seguinte, em produção, é o Kyverno exigir na admissão que a imagem
> tenha procedência verificável — política aplicada no cluster. Não vamos
> executá-la aqui porque exige um registry acessível pelo cluster, mas o
> conceito é o mesmo: imagem sem procedência não entra, ainda que passe em todo
> o restante."

---

## 👁️ Parte 4 — Demo 4: Runtime

> **Objetivo:** a admissão previne o previsível; o runtime detecta o restante —
> e evidencia o ataque do bloco 1 ocorrendo em tempo real.

### Passo 12: Observar os alertas do Falco (janela A)

**O que dizer:**
> "O Falco observa syscalls, não configuração. Deixo os alertas dele em execução
> nesta janela."

Em uma **janela A**, mantenha os alertas em execução:
```bash
kubectl -n falco logs -l app.kubernetes.io/name=falco -f | grep DOUGBR
```

> Mantenha esta janela visível. Abra a próxima em uma segunda janela.

### Passo 13: Reproduzir o ataque (janela B)

**O que dizer:**
> "Em um pod que já passou pelo portão de admissão, reproduzo o comportamento de
> um atacante: abro um shell e leio o token da ServiceAccount — o mesmo ataque
> do bloco 1."

Na **janela B**, aplique o pod-alvo:
```bash
kubectl apply -f demos/04-runtime/pod-vitima.yaml
```

Aguarde ficar pronto:
```bash
kubectl -n producao wait --for=condition=Ready pod/vitima --timeout=60s
```

Acesse o pod:
```bash
kubectl -n producao exec -it vitima -- sh
```

Dentro do pod — a leitura do token dispara o alerta:
```sh
cat /var/run/secrets/kubernetes.io/serviceaccount/token | head -c 40
```
```sh
exit
```

**Resultado esperado (na janela A):** em segundos, os alertas `[DOUGBR]`
aparecem — shell aberto no contêiner e leitura de arquivo sensível —, com pod,
comando e usuário.

> "A admissão observa o manifesto; o runtime observa a syscall. As duas são
> necessárias: a prevenção falha em silêncio, e a detecção é o que avisa que
> falhou. Um alerta sem alguém acompanhando é apenas registro."

### 🧹 Limpeza

```bash
kubectl -n producao delete pod vitima --ignore-not-found
```

---

## 🏁 Parte 5 — Encerramento

Apoio nos slides:

- Recapitulação da jornada `LAPTOP → CI/CD → CLUSTER`, com cada etapa cobrindo uma camada dos 4C's.
- A camada externa (Cloud): IAM, IMDS, KMS — a demonstração é local, produção não é.
- **Checklist de 10 itens** (o slide a fotografar):

> "Apenas os itens 1, 2 e 4 já neutralizam os dois vetores do bloco 1. Três
> configurações, nenhuma licença."

- Repositório, LinkedIn e DOUGBR.

### 🧹 Encerrar o ambiente (após a sessão)

Remova os componentes instalados (sem apagar o cluster do Docker Desktop):
```bash
kubectl delete ns producao dev ataque --ignore-not-found
```
```bash
helm -n kyverno uninstall kyverno; helm -n falco uninstall falco
```

> Para remover o cluster por completo: Docker Desktop › Settings › Kubernetes ›
> desmarque **Enable Kubernetes** (ou use **Reset Kubernetes Cluster**).

---

## 🔧 Solução de problemas

| Sintoma | Causa provável | Solução |
|---------|----------------|---------|
| Demo não sobe | imprevisto qualquer | Exibir o trecho da gravação de apoio e prosseguir |
| Falco sem alertas | filtro do grep restritivo | `kubectl -n falco logs -l app.kubernetes.io/name=falco --tail=50` sem grep |
| Falco em CrashLoop | driver eBPF | Reinstalar o chart (Passo 0.1) com `--set driver.kind=ebpf` |
| Kyverno não bloqueia | política não aplicada | `kubectl get clusterpolicy` — verificar `READY: True` |
| Pod `app` em `CreateContainerConfigError` | imagem nonroot sem UID numérico | O deploy já define `runAsUser: 65532`; reaplicar o manifesto |
| Pod `app` em `ImagePullBackOff` | imagem não carregada no cluster | `kind load docker-image dougbr/app:seguro --name desktop` |
| `gh run watch` sem retorno | execução ainda na fila | Aguardar alguns segundos ou usar `gh run view --web` |
| GitHub Actions indisponível | rede | Utilizar as capturas de tela do ensaio |

---

**Fim da masterclass.**
