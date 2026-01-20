# n8n-cli
flocked from https://github.com/edenreich/n8n-cli
enhance for VCG daily uses

## Download & Install GO 
```bash
https://go.dev/dl/
```
## Install Task
```bash
go install github.com/go-task/task/v3/cmd/task@latest
# verify
task --version
```

## Build and develope the n8n-cli tooling
```bash
cd n8n-cli
task install
```

## ENV Setting
```bash
./scripts/env_processing.sh -d -i .env.enc -o .env
cat .env >> ~/.bashrc

# optional, if set alias, then no need to type so long command
echo "alias n8n='n8n workflows' >> ~/.bashrc"

# run another bash session to make it effective
bash
```

## Usage
If alias set, then only need to type 'n8n' instead of 'n8n workflows'
```bash
# Initial current workflow
n8n workflows refresh -d n8n/
# Pull 1 workflow to local
n8n workflows pull -d n8n -n "WORKFLOW-NAME"
# or
n8n workflows pull -d n8n --id "XXXXX"

# Push the workflow to n8n
n8n workflows push -f n8n/XXXX.json
```

