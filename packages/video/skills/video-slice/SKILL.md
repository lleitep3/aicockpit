---
name: video-slice
description: "Slices video files into images for cost-effective analysis before processing."
---

# Habilidade: video-slice

Esta habilidade ensina o agente a interceptar solicitações de análise de vídeo, fatiar o vídeo usando a ferramenta CLI do Cockpit e analisar as imagens geradas para economizar tokens.

## Como Usar

Quando o usuário pedir para descrever, depurar ou analisar um vídeo:

1. **Execute o fatiamento no terminal:**
   ```bash
   cockpit video slice "/caminho/do/video.mp4" --interval 5
   ```
   *Nota: O comando padrão salvará as imagens em `~/.cockpit/workspace/video-slice/<video-name>/slices`.*

2. **Acesse os frames gerados:**
   * Liste os arquivos do diretório usando a ferramenta de listagem de diretório para identificar os frames extraídos (`frame_0001.jpg`, `frame_0002.jpg`, etc).
   * Utilize a ferramenta de visualização de arquivos (`view_file`) para ver e analisar as imagens mais relevantes para o pedido do usuário.

3. **Formule a resposta baseando-se nos frames:**
   * Analise o fluxo do vídeo frame a frame a partir das imagens.
   * Explique as ações e eventos que ocorrem.
   * Se necessário, sugira ao usuário o comando ffmpeg exato caso ele queira fatiar vídeos fora do ambiente.
