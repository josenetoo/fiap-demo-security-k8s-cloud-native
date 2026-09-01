# Laboratório — Segurança em Kubernetes com Cloud Native

**Evento:** DOUGBR — DevOps User Group Brazil
**Instrutor:** José Neto — Cloud Solution Engineer & Tech Lead · Coordenador da PosTech de DevOps e Arquitetura Cloud da FIAP
**Temas:** 4C's · Ataque a um cluster · Trivy + CI/CD · Admissão (PSA + Kyverno) · Runtime (Falco)
**Formato:** quatro demonstrações na jornada `LAPTOP → CI/CD → CLUSTER`

---

## Antes de começar

Este laboratório exige Docker Desktop com Kubernetes habilitado. Os comandos de Docker, kubectl, kind, Helm, Trivy, jq e GitHub CLI são iguais no macOS, Linux e Windows. Quando a sintaxe do shell muda, o documento apresenta blocos separados para:

- macOS com Zsh ou Bash;
- Linux com Bash;
- Windows com PowerShell.

Cada bloco contém um único comando para facilitar a execução durante a demonstração.

### Base: Docker Desktop

1. Instale e abra o Docker Desktop.
2. Em **Settings › Resources › Memory**, defina **8 GB**.
3. Em **Settings › Kubernetes**, marque **Enable Kubernetes** e selecione **Apply & Restart**.
4. Aguarde o indicador do Kubernetes ficar verde.

Confirme o Docker em qualquer sistema:

```console
docker version
```

Confirme o contexto atual:

```console
kubectl config current-context
```

Confirme os nós:

```console
kubectl get nodes
```

**Resultado esperado:** contexto `docker-desktop` e três nós `Ready`: `desktop-control-plane`, `desktop-worker` e `desktop-worker2`.

O nome do cluster usado pelo `kind load` é `desktop`. Não é necessário criar outro cluster.

### Verificar as ferramentas

Execute cada comando no macOS, Linux ou Windows:

```console
kind version
```

```console
helm version
```

```console
trivy version
```

```console
jq --version
```

```console
gh --version
```

### Instalar no macOS

Com Homebrew:

```bash
brew install kind
```

```bash
brew install helm
```

```bash
brew install trivy
```

```bash
brew install jq
```

```bash
brew install gh
```

### Instalar no Linux

Com Homebrew for Linux:

```bash
brew install kind
```

```bash
brew install helm
```

```bash
brew install trivy
```

```bash
brew install jq
```

```bash
brew install gh
```

Em distribuições sem Homebrew, instale os mesmos cinco binários pelos repositórios oficiais de cada projeto.

### Instalar no Windows

Com PowerShell e winget:

```powershell
winget install --id Kubernetes.kind --exact
```

```powershell
winget install --id Helm.Helm --exact
```

```powershell
winget install --id AquaSecurity.Trivy --exact
```

```powershell
winget install --id jqlang.jq --exact
```

```powershell
winget install --id GitHub.cli --exact
```

### Autenticar a GitHub CLI

Este comando é igual nos três sistemas:

```console
gh auth login
```

---

## Parte 0 — Setup

As imagens permanecem locais. O Trivy as analisa no Docker, e o `kind load` as carrega no cluster sem usar um registry.

### Passo 0.1 — Instalar Kyverno e Falco

Adicione o repositório do Kyverno:

```console
helm repo add kyverno https://kyverno.github.io/kyverno/
```

Atualize os repositórios do Helm:

```console
helm repo update
```

Instale o Kyverno:

```console
helm upgrade --install kyverno kyverno/kyverno --namespace kyverno --create-namespace --wait --timeout 5m
```

Adicione o repositório do Falco:

```console
helm repo add falcosecurity https://falcosecurity.github.io/charts
```

Atualize os repositórios do Helm:

```console
helm repo update
```

#### Configurar o webhook do Slack no macOS e Linux

Use o `export` padrão e substitua o valor pelo webhook criado no Slack:

```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/SEU/WEBHOOK/AQUI"
```

Confirme apenas que a variável possui um valor, sem imprimir a credencial:

```bash
test -n "$SLACK_WEBHOOK_URL" && echo "SLACK_WEBHOOK_URL configurada"
```

#### Configurar o webhook do Slack no Windows

No PowerShell, defina a variável de ambiente da sessão:

```powershell
$env:SLACK_WEBHOOK_URL = "https://hooks.slack.com/services/SEU/WEBHOOK/AQUI"
```

Confirme apenas que a variável possui um valor, sem imprimir a credencial:

```powershell
if ($env:SLACK_WEBHOOK_URL) { Write-Host "SLACK_WEBHOOK_URL configurada" }
```

O webhook é uma credencial. Não salve o valor no repositório. O comando pode ficar no histórico local do shell; remova o histórico se a máquina for compartilhada e revogue o webhook após a sessão.

#### Instalar o Falco no macOS e Linux

```bash
helm upgrade --install falco falcosecurity/falco -n falco --create-namespace \
  --set driver.kind=modern_ebpf \
  --set tty=true \
  --set-file 'customRules.dougbr-rules\.yaml'=demos/04-runtime/regras-dougbr.yaml \
  --set falco.json_output=true \
  --set falco.http_output.enabled=true \
  --set falco.http_output.url=http://falco-falcosidekick:2801 \
  --set falcosidekick.enabled=true \
  --set-string falcosidekick.config.slack.webhookurl="$SLACK_WEBHOOK_URL" \
  --set falcosidekick.config.slack.minimumpriority=warning
```

#### Instalar o Falco no Windows

No PowerShell, o caractere de continuação é o acento grave:

```powershell
helm upgrade --install falco falcosecurity/falco -n falco --create-namespace `
  --set driver.kind=modern_ebpf `
  --set tty=true `
  --set-file 'customRules.dougbr-rules\.yaml'=demos/04-runtime/regras-dougbr.yaml `
  --set falco.json_output=true `
  --set falco.http_output.enabled=true `
  --set falco.http_output.url=http://falco-falcosidekick:2801 `
  --set falcosidekick.enabled=true `
  --set-string "falcosidekick.config.slack.webhookurl=$env:SLACK_WEBHOOK_URL" `
  --set falcosidekick.config.slack.minimumpriority=warning
```

#### Instalar sem integração com o Slack

Este comando é igual nos três sistemas:

```console
helm upgrade --install falco falcosecurity/falco -n falco --create-namespace --set driver.kind=modern_ebpf --set tty=true --set-file customRules.dougbr-rules\.yaml=demos/04-runtime/regras-dougbr.yaml --set falco.json_output=false
```

Se o kernel não oferecer BTF, reinstale usando o driver `ebpf`. Execute a variante de instalação do seu sistema trocando somente o valor do driver por:

```text
--set driver.kind=ebpf
```

Confirme os DaemonSets:

```console
kubectl -n falco get ds
```

Confirme os pods:

```console
kubectl -n falco get pods
```

**Resultado esperado:** o DaemonSet do Falco com todos os nós prontos, os pods `falco-*` em `2/2 Running` e, quando habilitado, o Falcosidekick em execução.

Instale o Falco antes das políticas do Kyverno da Parte 3. O DaemonSet requer `privileged` e `hostPath`; as políticas do laboratório excluem namespaces de infraestrutura, mas esta ordem simplifica a instalação.

### Passo 0.2 — Construir e carregar as imagens

Construa a imagem insegura:

```console
docker build -f demos/02-build/Dockerfile.inseguro -t dougbr/app:inseguro demos/02-build
```

Construa a imagem segura:

```console
docker build -f demos/02-build/Dockerfile.seguro -t dougbr/app:seguro demos/02-build
```

Carregue a imagem segura no cluster:

```console
kind load docker-image dougbr/app:seguro --name desktop
```

Carregue a imagem insegura no cluster:

```console
kind load docker-image dougbr/app:inseguro --name desktop
```

### Passo 0.3 — Confirmar o ambiente

Confirme os nós:

```console
kubectl get nodes
```

Confirme o Kyverno:

```console
kubectl -n kyverno get pods
```

Confirme o Falco:

```console
kubectl -n falco get pods
```

No macOS e Linux, confirme as imagens:

```bash
docker images | grep 'dougbr/app'
```

No Windows, confirme as imagens:

```powershell
docker images | Select-String 'dougbr/app'
```

**Resultado esperado:** três nós `Ready`, pods do Kyverno e do Falco em execução e as imagens `dougbr/app:seguro` e `dougbr/app:inseguro` no Docker.

A publicação no GitHub não faz parte do setup. Publique o repositório quando desejar; ele precisa estar disponível no GitHub somente antes do Passo 6.

---

## Parte 1 — Demo 1: ataque

**Objetivo:** demonstrar dois caminhos de elevação de privilégio quando o usuário já possui permissão para criar recursos perigosos: acesso ao host por um pod privilegiado e permissões excessivas por RBAC.

### Passo 1 — Acesso ao host por pod privilegiado

Crie o namespace:

```console
kubectl apply -f demos/01-ataque/00-namespace.yaml
```

Aplique o pod privilegiado:

```console
kubectl -n ataque apply -f demos/01-ataque/10-pod-privilegiado.yaml
```

Aguarde o pod ficar pronto:

```console
kubectl -n ataque wait --for=condition=Ready pod/pod-privilegiado --timeout=60s
```

Acesse o pod:

```console
kubectl -n ataque exec -it pod-privilegiado -- sh
```

Os próximos comandos são executados dentro do contêiner e são iguais em todos os sistemas.

Liste os arquivos de configuração do Kubernetes no nó:

```sh
ls -la /host/etc/kubernetes/
```

Leia o início do kubeconfig administrativo:

```sh
cat /host/etc/kubernetes/admin.conf | head -20
```

Liste os arquivos da PKI:

```sh
ls -la /host/etc/kubernetes/pki/
```

Saia do contêiner:

```sh
exit
```

**Resultado esperado:** no cluster local, o arquivo `admin.conf` e o diretório `pki/` do control plane ficam acessíveis pelo contêiner.

### Passo 2 — Permissões excessivas por ServiceAccount

Crie a ServiceAccount e o ClusterRoleBinding permissivo:

```console
kubectl apply -f demos/01-ataque/20-rbac-generoso.yaml
```

Aplique o pod comum:

```console
kubectl apply -f demos/01-ataque/30-pod-comum.yaml
```

Aguarde o pod ficar pronto:

```console
kubectl -n ataque wait --for=condition=Ready pod/pod-comum --timeout=60s
```

Acesse o pod:

```console
kubectl -n ataque exec -it pod-comum -- sh
```

Os próximos comandos são executados dentro do contêiner.

Leia o token da ServiceAccount:

```sh
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
```

Instale o curl no contêiner:

```sh
apk add --no-cache curl >/dev/null 2>&1
```

Consulte os secrets pela API do Kubernetes:

```sh
curl -sk -H "Authorization: Bearer $TOKEN" https://kubernetes.default.svc/api/v1/secrets | head -30
```

Saia do contêiner:

```sh
exit
```

**Resultado esperado:** a API retorna a lista de secrets porque a ServiceAccount está vinculada a `cluster-admin`.

### Limpeza da Parte 1

Remova o namespace:

```console
kubectl delete ns ataque --ignore-not-found
```

Remova o ClusterRoleBinding:

```console
kubectl delete clusterrolebinding app-legado-cluster-admin --ignore-not-found
```

---

## Parte 2 — Demo 2: build e CI/CD

**Objetivo:** analisar imagens, configuração e segredos antes do cluster e repetir as verificações no CI.

### Passo 3 — Base completa versus distroless

Analise a imagem insegura:

```console
trivy image --scanners vuln --severity HIGH,CRITICAL dougbr/app:inseguro
```

Analise a imagem segura:

```console
trivy image --scanners vuln --severity HIGH,CRITICAL dougbr/app:seguro
```

**Resultado esperado:** a quantidade exata varia com a data e com o banco de vulnerabilidades. A imagem distroless deve apresentar menos pacotes e uma superfície menor que a imagem baseada na distribuição completa.

### Passo 4 — Configuração e segredos

Analise a configuração do Dockerfile inseguro:

```console
trivy config demos/02-build/Dockerfile.inseguro
```

Analise os segredos usados na demonstração:

```console
trivy fs --scanners secret demos/02-build/
```

**Resultado esperado:** o Trivy aponta a ausência de `USER` e identifica a credencial AWS fictícia inserida intencionalmente no Dockerfile inseguro.

### Passo 5 — Gerar o SBOM

Gere o inventário no diretório atual:

```console
trivy image --format cyclonedx --output sbom.json dougbr/app:seguro
```

Conte os componentes catalogados:

```console
jq '.components | length' sbom.json
```

**Resultado esperado:** o comando retorna o total de componentes identificados na imagem. O arquivo `sbom.json` está no `.gitignore`.

### Passo 6 — Pipeline de CI

Antes deste passo, publique o repositório no GitHub e confirme a autenticação:

```console
gh auth status
```

#### Execução aprovada

Dispare manualmente o workflow aprovado:

```console
gh workflow run security.yml
```

Acompanhe a execução:

```console
gh run watch
```

Para abrir os jobs no navegador:

```console
gh run view --web
```

**Resultado esperado:** `code-scan` e `build-scan` concluem com sucesso e o SBOM é publicado como artefato. Os scans de configuração e segredos bloqueiam o workflow; nesta demonstração, o scan de vulnerabilidades da imagem é informativo.

#### Execução reprovada

Dispare o workflow que usa os artefatos inseguros:

```console
gh workflow run security-fail-demo.yml
```

Acompanhe a execução:

```console
gh run watch
```

**Resultado esperado:** o job falha nos controles de configuração e segredo.

A falha do workflow só impede merge quando o repositório possui uma branch protection ou um ruleset que exige esse check.

---

## Parte 3 — Demo 3: admissão

**Objetivo:** bloquear configurações perigosas mesmo quando um recurso é aplicado diretamente no cluster.

### Passo 7 — Pod Security Admission no namespace `producao`

Crie o namespace com o perfil `restricted`:

```console
kubectl apply -f demos/03-admission/policies/00-psa-labels.yaml
```

Tente aplicar o pod privilegiado:

```console
kubectl -n producao apply -f demos/01-ataque/10-pod-privilegiado.yaml
```

**Resultado esperado:** o Pod Security Admission recusa o pod com a mensagem `violates PodSecurity "restricted:latest"`.

### Passo 8 — Kyverno no namespace `dev`

Crie o namespace sem labels de PSA:

```console
kubectl apply -f demos/03-admission/policies/05-namespace-dev.yaml
```

Aplique a política que bloqueia contêineres privilegiados:

```console
kubectl apply -f demos/03-admission/policies/10-bloquear-privilegiado.yaml
```

Aplique a política que bloqueia `hostPath`:

```console
kubectl apply -f demos/03-admission/policies/20-bloquear-hostpath.yaml
```

Aplique a política que exige execução como não root:

```console
kubectl apply -f demos/03-admission/policies/30-exigir-nonroot.yaml
```

Confira as políticas:

```console
kubectl get clusterpolicy
```

Tente aplicar o pod privilegiado em `dev`:

```console
kubectl -n dev apply -f demos/01-ataque/10-pod-privilegiado.yaml
```

**Resultado esperado:** o Kyverno bloqueia o pod e retorna as mensagens definidas pelas políticas.

As políticas excluem `kube-system`, `kyverno` e `falco` porque componentes de infraestrutura podem precisar de privilégios e volumes do host.

### Passo 9 — Mutação de defaults seguros

Aplique a política de mutação:

```console
kubectl apply -f demos/03-admission/policies/40-mutar-defaults-seguros.yaml
```

No macOS e Linux, consulte o `securityContext` admitido:

```bash
kubectl -n dev run teste --image=dougbr/app:seguro --dry-run=server -o yaml | grep -A6 securityContext
```

No Windows, consulte o `securityContext` admitido:

```powershell
kubectl -n dev run teste --image=dougbr/app:seguro --dry-run=server -o yaml | Select-String -Pattern 'securityContext' -Context 0,6
```

**Resultado esperado:** o Kyverno injeta `runAsNonRoot`, remoção de capabilities e seccomp.

### Passo 10 — Deploy compatível com as políticas

Aplique o deployment:

```console
kubectl apply -f demos/03-admission/90-deploy-seguro.yaml
```

Acompanhe os pods:

```console
kubectl -n producao get pods -w
```

Interrompa o acompanhamento com `Ctrl+C` quando os pods estabilizarem.

**Resultado esperado:** os pods `app` alcançam `Running` com `1/1` contêiner pronto e usam a imagem carregada pelo `kind load`.

### Passo 11 — Próximo controle de supply chain

Este laboratório não aplica política de assinatura, atestação ou proveniência de imagens. Esse controle exige uma imagem publicada e uma política compatível com o registry adotado.

---

## Parte 4 — Demo 4: runtime

**Objetivo:** detectar a abertura de shell e a leitura do token de uma ServiceAccount depois que o pod já foi admitido.

### Passo 12 — Observar os alertas do Falco

No macOS e Linux, acompanhe as mensagens das regras do laboratório:

```bash
kubectl -n falco logs -l app.kubernetes.io/name=falco -f | grep --line-buffered DOUGBR | jq -r --unbuffered '.output'
```

No Windows, acompanhe as mensagens pelo PowerShell:

```powershell
kubectl -n falco logs -l app.kubernetes.io/name=falco -f | Select-String 'DOUGBR' | ForEach-Object { $_.Line } | jq -r '.output'
```

Se o filtro não produzir saída, consulte os logs brutos em qualquer sistema:

```console
kubectl -n falco logs -l app.kubernetes.io/name=falco --tail=50
```

### Passo 13 — Gerar os eventos de runtime

Em outro terminal, aplique o pod-alvo:

```console
kubectl apply -f demos/04-runtime/pod-observado.yaml
```

Aguarde o pod ficar pronto:

```console
kubectl -n producao wait --for=condition=Ready pod/observado --timeout=60s
```

Acesse o pod:

```console
kubectl -n producao exec -it observado -- sh
```

Os próximos comandos são executados dentro do contêiner.

Leia parte do token:

```sh
cat /var/run/secrets/kubernetes.io/serviceaccount/token | head -c 40
```

Saia do contêiner:

```sh
exit
```

**Resultado esperado:** os alertas `[DOUGBR]` registram a abertura do shell e a leitura do arquivo sensível. Quando o Slack está configurado, os eventos também chegam ao canal definido.

Consulte o encaminhamento do Falcosidekick:

```console
kubectl -n falco logs -l app.kubernetes.io/name=falcosidekick --tail=20
```

### Limpeza da Parte 4

Remova o pod:

```console
kubectl -n producao delete pod observado --ignore-not-found
```

---

## Encerrar o ambiente

Remova os namespaces:

```console
kubectl delete ns producao dev ataque --ignore-not-found
```

Remova o Kyverno:

```console
helm -n kyverno uninstall kyverno
```

Remova o Falco:

```console
helm -n falco uninstall falco
```

No macOS e Linux, remova a variável da sessão:

```bash
unset SLACK_WEBHOOK_URL
```

No Windows, remova a variável da sessão:

```powershell
Remove-Item Env:SLACK_WEBHOOK_URL
```

Para remover o cluster, desmarque **Enable Kubernetes** ou use **Reset Kubernetes Cluster** nas configurações do Docker Desktop.

---

## Solução de problemas

### Falco sem alertas

Consulte os logs sem filtro:

```console
kubectl -n falco logs -l app.kubernetes.io/name=falco --tail=50
```

### Falco em `CrashLoopBackOff`

Consulte os pods:

```console
kubectl -n falco get pods
```

Reinstale usando a variante do comando Helm do seu sistema e altere o driver para:

```text
--set driver.kind=ebpf
```

### DaemonSet do Falco não é criado

Confirme as políticas e suas exclusões:

```console
kubectl get clusterpolicy
```

Depois de corrigir as exclusões, reinstale o Falco com o comando do Passo 0.1.

### Alerta no terminal, mas não no Slack

No macOS e Linux, confira a saída HTTP:

```bash
kubectl -n falco get cm falco -o yaml | grep -A3 http_output
```

No Windows, confira a saída HTTP:

```powershell
kubectl -n falco get cm falco -o yaml | Select-String -Pattern 'http_output' -Context 0,3
```

Recrie os pods do Falco:

```console
kubectl -n falco delete pod -l app.kubernetes.io/name=falco
```

### Falcosidekick sem eventos

Consulte os logs do Falcosidekick:

```console
kubectl -n falco logs -l app.kubernetes.io/name=falcosidekick --tail=20
```

### Kyverno não bloqueia

Confirme se as políticas estão prontas:

```console
kubectl get clusterpolicy
```

### Pod `app` em `CreateContainerConfigError`

Confirme o UID numérico no deployment:

```console
kubectl -n producao get deployment app -o yaml
```

### Pod `app` em `ImagePullBackOff`

Carregue novamente a imagem:

```console
kind load docker-image dougbr/app:seguro --name desktop
```

### GitHub Actions em fila

Consulte a execução no navegador:

```console
gh run view --web
```
