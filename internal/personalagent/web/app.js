let token = document.querySelector('meta[name="cockpit-token"]').content;
const state = {
  providers: [], catalogs: new Map(), projects: [], activeProjectId: "", selectedProjectIds: new Set(),
  renamingProjectId: "", editingProjectId: "", defaultWorkspace: "",
  conversations: [], conversation: null, collapsedRootIds: new Set(), showArchived: false,
  activeTurn: null, activeTurnConversationID: "", eventIDs: new Map(), attachments: [], routing: null,
  unreadConversations: new Set(), conversationSyncInitialized: false, conversationSyncInProgress: false,
  conversationSyncErrorShown: false, sendingMessage: false, addingAttachments: false, dropDepth: 0, markdownWarningShown: false,
  draft: { provider_id: "antigravity", model_id: "", reasoning_effort: "", approval_policy: "on-request", routing_mode: "manual" },
};
const byId = (id) => document.getElementById(id);
const routeClasses = [
  { id: "quick", label: "Rápida", description: "Pedidos diretos, curtos e reformulações." },
  { id: "coding", label: "Código", description: "Implementar, corrigir, testar ou revisar código." },
  { id: "research", label: "Pesquisa", description: "Informação atual, fontes e pesquisa externa." },
  { id: "reasoning", label: "Raciocínio", description: "Planos, arquitetura, comparação e análise." },
];

async function refreshToken() {
  try {
    const html = await fetch("/").then((r) => r.text());
    const match = html.match(/name="cockpit-token"\s+content="([^"]+)"/);
    if (match?.[1]) {
      token = match[1];
      return true;
    }
  } catch {
    // ignore
  }
  return false;
}

async function api(path, options = {}, retried = false) {
  const headers = new Headers(options.headers || {});
  headers.set("Authorization", `Bearer ${token}`);
  if (options.body) headers.set("Content-Type", "application/json");
  const response = await fetch(`/api/v1${path}`, { ...options, headers });
  if (response.status === 401 && !retried && (await refreshToken())) {
    return api(path, options, true);
  }
  const payload = response.headers.get("content-type")?.includes("application/json") ? await response.json() : null;
  if (!response.ok) throw new Error(payload?.error || `Falha na requisição (${response.status})`);
  return payload;
}

function showToast(message) {
  const toast = byId("toast");
  toast.textContent = message;
  toast.classList.add("visible");
  clearTimeout(showToast.timer);
  showToast.timer = setTimeout(() => toast.classList.remove("visible"), 3600);
}

function makeOption(label, value, disabled = false) {
  const item = document.createElement("option");
  item.value = value;
  item.textContent = label;
  item.disabled = disabled;
  return item;
}

function currentSelection() {
  return state.conversation?.selection || state.draft;
}

async function loadProviders() {
  const result = await api("/providers");
  state.providers = result.providers;
  const preferred = state.providers.find((p) => p.id === "antigravity" && p.supported)
    || state.providers.find((p) => p.id === "copilot" && p.supported)
    || state.providers.find((p) => p.supported);
  const defaultProviderId = preferred?.id || "codex";
  state.draft.provider_id = defaultProviderId;
  await loadModels(defaultProviderId);
  const first = state.catalogs.get(defaultProviderId)?.models.find((model) => model.default && model.selectable)
    || state.catalogs.get(defaultProviderId)?.models.find((model) => model.selectable);
  if (first) state.draft.model_id = first.id;
  renderProviderOptions();
  updateSelectionControls();
}

async function loadModels(providerId, force = false) {
  if (!force && state.catalogs.has(providerId)) return state.catalogs.get(providerId);
  const result = await api(`/providers/${encodeURIComponent(providerId)}/models`);
  state.catalogs.set(providerId, result);
  if (currentSelection().provider_id === providerId) renderModelOptions();
  return result;
}

async function loadProjects() {
  const result = await api("/projects");
  state.projects = result.projects || [];
  renderProjectList();
  updateTopbarProjectState();
}

function renderProjectList() {
  const list = byId("project-list");
  if (!list) return;
  list.replaceChildren();
  if (!state.projects.length) {
    const empty = document.createElement("div");
    empty.className = "project-empty";
    empty.textContent = "Nenhum projeto criado";
    list.append(empty);
    return;
  }
  const statsByProject = window.CockpitInteractions?.projectConversationStats
    ? window.CockpitInteractions.projectConversationStats(
        state.projects,
        state.conversations,
        state.unreadConversations,
        state.showArchived,
      )
    : new Map();
  for (const project of state.projects) {
    const isChecked = state.selectedProjectIds.has(project.id);
    const stats = statsByProject.get(project.id) || { count: 0, hasUpdate: false, hasRunning: false };
    const row = document.createElement("div");
    row.className = `project-item${isChecked ? " selected" : ""}`;

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.className = "project-checkbox";
    checkbox.checked = isChecked;
    checkbox.title = "Filtrar conversas deste projeto";
    checkbox.setAttribute("aria-label", `Filtrar projeto ${project.name}`);
    checkbox.addEventListener("click", (event) => event.stopPropagation());
    checkbox.addEventListener("change", () => toggleProjectFilter(project.id, checkbox.checked));
    row.append(checkbox);

    if (state.renamingProjectId === project.id) {
      const input = document.createElement("input");
      input.type = "text";
      input.className = "project-rename-input";
      input.value = project.name || "Novo projeto";
      input.setAttribute("aria-label", "Nome do projeto");
      let committed = false;
      const commit = async (cancel = false) => {
        if (committed) return;
        committed = true;
        state.renamingProjectId = "";
        const nextName = input.value.trim();
        if (cancel || !nextName || nextName === project.name) {
          renderProjectList();
          return;
        }
        await renameProjectInline(project, nextName);
      };
      input.addEventListener("keydown", (event) => {
        if (event.key === "Enter") { event.preventDefault(); commit(false); }
        if (event.key === "Escape") { event.preventDefault(); commit(true); }
      });
      input.addEventListener("blur", () => commit(false));
      row.append(input);
      list.append(row);
      requestAnimationFrame(() => { input.focus(); input.select(); });
      continue;
    }

    const main = document.createElement("button");
    main.type = "button";
    main.className = "project-item-main";
    const nameRow = document.createElement("div");
    nameRow.className = "project-name-row";
    const name = document.createElement("span");
    name.className = "project-name";
    name.textContent = project.name || "Novo projeto";
    name.title = "Clique para marcar/desmarcar filtro · Duplo clique para renomear";
    nameRow.append(name);
    if (stats.hasUpdate) {
      const updateDot = document.createElement("span");
      updateDot.className = `project-update-dot${stats.hasRunning ? " is-running" : ""}`;
      updateDot.title = stats.hasRunning ? "Conversa em execução neste projeto" : "Conversas atualizadas neste projeto";
      updateDot.setAttribute("aria-label", updateDot.title);
      nameRow.append(updateDot);
    }
    main.append(nameRow);
    if (project.workspace || project.repositories?.length || project.task_count) {
      const meta = document.createElement("small");
      meta.className = "project-meta";
      const parts = [];
      if (project.workspace) parts.push(project.workspace.split("/").filter(Boolean).slice(-2).join("/"));
      if (project.repositories?.length) parts.push(`${project.repositories.length} repo(s)`);
      if (project.task_count) parts.push(`${project.task_count} task(s)`);
      meta.textContent = parts.join(" · ");
      main.append(meta);
    }
    main.addEventListener("click", () => toggleProjectFilter(project.id));
    main.addEventListener("dblclick", (event) => {
      event.stopPropagation();
      state.renamingProjectId = project.id;
      renderProjectList();
    });

    const countBadge = document.createElement("span");
    countBadge.className = "project-count-badge";
    countBadge.textContent = String(stats.count);
    countBadge.title = `${stats.count} conversa(s)`;

    const infoBtn = document.createElement("button");
    infoBtn.type = "button";
    infoBtn.className = "project-info-btn";
    infoBtn.title = "Informações e configurações do projeto";
    infoBtn.setAttribute("aria-label", `Configurar ${project.name}`);
    infoBtn.textContent = "⚙";
    infoBtn.addEventListener("click", (event) => {
      event.stopPropagation();
      openProjectModal(project.id);
    });

    row.append(main, countBadge, infoBtn);
    list.append(row);
  }
}

async function createInlineProject() {
  try {
    const project = await api("/projects", {
      method: "POST",
      body: JSON.stringify({ name: "Novo projeto" }),
    });
    state.selectedProjectIds.add(project.id);
    state.activeProjectId = project.id;
    state.renamingProjectId = project.id;
    await loadProjects();
    renderConversationList();
    updateTopbarProjectState();
  } catch (error) {
    showToast(error.message);
  }
}

async function renameProjectInline(project, nextName) {
  try {
    await api(`/projects/${encodeURIComponent(project.id)}`, {
      method: "PATCH",
      body: JSON.stringify({
        name: nextName,
        description: project.description || "",
        workspace: project.workspace || "",
        repositories: project.repositories || [],
        instructions: project.instructions || "",
        provider_id: project.provider_id || "",
        model_id: project.model_id || "",
        reasoning_effort: project.reasoning_effort || "",
        approval_policy: project.approval_policy || "",
        routing_mode: project.routing_mode || "",
      }),
    });
    await loadProjects();
    renderConversationList();
  } catch (error) {
    showToast(error.message);
    renderProjectList();
  }
}

function toggleProjectFilter(projectId, forceState) {
  if (!projectId) {
    state.selectedProjectIds.clear();
    state.activeProjectId = "";
  } else {
    const shouldCheck = typeof forceState === "boolean" ? forceState : !state.selectedProjectIds.has(projectId);
    if (shouldCheck) {
      state.selectedProjectIds.add(projectId);
      state.activeProjectId = projectId;
    } else {
      state.selectedProjectIds.delete(projectId);
      if (state.activeProjectId === projectId) {
        state.activeProjectId = Array.from(state.selectedProjectIds).pop() || "";
      }
    }
  }
  const activeProject = state.projects.find((item) => item.id === state.activeProjectId);
  if (activeProject && !state.conversation) {
    applyProjectDefaultsToDraft(activeProject);
  }
  renderProjectList();
  renderConversationList();
  updateTopbarProjectState();
}

function selectProject(projectId) {
  toggleProjectFilter(projectId);
}

function applyProjectDefaultsToDraft(project) {
  if (!project) return;
  if (project.provider_id) state.draft.provider_id = project.provider_id;
  if (project.model_id) state.draft.model_id = project.model_id;
  if (project.reasoning_effort != null) state.draft.reasoning_effort = project.reasoning_effort;
  if (project.approval_policy) state.draft.approval_policy = project.approval_policy;
  if (project.routing_mode) state.draft.routing_mode = project.routing_mode;
  updateSelectionControls();
}

function updateTopbarProjectState() {
  const activeProjectId = state.conversation?.project_id || state.activeProjectId;
  const project = state.projects.find((item) => item.id === activeProjectId);
  const pill = byId("project-active-pill");
  const clearBtn = byId("clear-project-filter");
  const sectionLabel = byId("conversations-section-label");
  const archiveBtn = byId("toggle-archived-conversations");
  const workspaceEl = byId("workspace");
  const assignSelect = byId("conversation-project-select");
  const assistantMode = Boolean(state.conversation?.assistant_mode);

  if (workspaceEl) {
    workspaceEl.textContent = assistantMode
      ? "Assistente · sandbox somente leitura"
      : state.conversation?.workspace || project?.workspace || state.defaultWorkspace || "";
  }
  if (pill) {
    pill.classList.toggle("hidden", !project);
    if (project) {
      byId("project-active-name").textContent = project.name;
      byId("project-active-edit").onclick = () => openProjectModal(project.id);
    }
  }
  if (clearBtn && sectionLabel) {
    const filtering = state.selectedProjectIds.size > 0;
    clearBtn.classList.toggle("hidden", !filtering);
    let label = state.showArchived ? "ARQUIVADAS" : "CONVERSAS";
    if (state.selectedProjectIds.size === 1) {
      const [onlyId] = state.selectedProjectIds;
      const filterProject = state.projects.find((item) => item.id === onlyId);
      if (filterProject) label = `${label} · ${filterProject.name.toUpperCase()}`;
    } else if (state.selectedProjectIds.size > 1) {
      label = `${label} · ${state.selectedProjectIds.size} PROJETOS`;
    }
    sectionLabel.textContent = label;
  }
  if (archiveBtn) {
    const archivedCount = state.conversations.filter((item) => item.archived).length;
    archiveBtn.classList.toggle("active-filter", state.showArchived);
    archiveBtn.setAttribute("aria-pressed", String(state.showArchived));
    archiveBtn.textContent = state.showArchived
      ? "← Ativas"
      : archivedCount > 0 ? `Arquivadas (${archivedCount})` : "Arquivadas";
  }
  if (assignSelect) {
    assignSelect.classList.toggle("hidden", !state.conversation || assistantMode);
    assignSelect.replaceChildren(makeOption("Sem projeto", ""));
    for (const item of state.projects) {
      assignSelect.append(makeOption(`📁 ${item.name}`, item.id));
    }
    assignSelect.value = state.conversation?.project_id || "";
  }

  const rootPill = byId("root-conversation-pill");
  const rootNameEl = byId("root-conversation-name");
  const parentSelect = byId("conversation-parent-select");
  const spawnBtn = byId("spawn-subchat-btn");
  const currentConv = state.conversation;
  const currentRootID = currentConv?.root_conversation_id || currentConv?.parent_conversation_id || "";
  const rootConv = currentRootID ? state.conversations.find((item) => item.id === currentRootID) : null;
  const childConvs = currentConv
    ? state.conversations.filter(
        (item) =>
          item.id !== currentConv.id &&
          (item.root_conversation_id === currentConv.id || item.parent_conversation_id === currentConv.id),
      )
    : [];

  if (rootPill && rootNameEl) {
    if (currentConv && currentRootID) {
      rootPill.classList.remove("hidden");
      const label = rootConv?.title || currentRootID.slice(0, 12);
      rootNameEl.textContent = `Raiz: ${label}`;
      rootPill.title = `Abrir conversa raiz: ${rootConv?.title || currentRootID}`;
      rootPill.onclick = () => openConversation(currentRootID);
    } else if (currentConv && childConvs.length > 0) {
      rootPill.classList.remove("hidden");
      rootNameEl.textContent = `Conversa raiz · ${childConvs.length} sub-sessão(ões)`;
      rootPill.title = "Esta é a conversa raiz deste grupo de sessões";
      rootPill.onclick = () => openConversation(childConvs[0].id);
    } else {
      rootPill.classList.add("hidden");
    }
  }

  if (spawnBtn) {
    spawnBtn.classList.toggle("hidden", !currentConv || assistantMode);
  }

  if (parentSelect) {
    parentSelect.classList.toggle("hidden", !currentConv || assistantMode);
    parentSelect.replaceChildren(makeOption("⑂ Sem conversa raiz (Independente)", ""));
    const candidateRoots = state.conversations.filter(
      (item) =>
        !item.assistant_mode &&
        item.id !== currentConv?.id &&
        item.root_conversation_id !== currentConv?.id &&
        item.parent_conversation_id !== currentConv?.id,
    );
    for (const item of candidateRoots) {
      const prefix = item.root_conversation_id ? "↳ " : "⑂ ";
      parentSelect.append(makeOption(`${prefix}${item.title || item.id}`, item.id));
    }
    parentSelect.value = currentConv?.parent_conversation_id || currentConv?.root_conversation_id || "";
  }
}

async function openProjectModal(projectId = "", focusAI = false) {
  state.editingProjectId = projectId;
  const project = state.projects.find((item) => item.id === projectId) || null;
  byId("project-modal-title").textContent = project ? `Projeto · ${project.name}` : "Criar projeto";
  byId("project-name-input").value = project?.name || "Novo projeto";
  byId("project-workspace-input").value = project?.workspace || "";
  byId("project-repos-input").value = (project?.repositories || []).join("\n");
  byId("project-description-input").value = project?.description || "";
  byId("project-instructions-input").value = project?.instructions || "";
  byId("project-routing-select").value = project?.routing_mode || "";
  byId("project-approval-select").value = project?.approval_policy || "";
  byId("project-ai-brief").value = "";
  byId("project-delete-btn").classList.toggle("hidden", !project);

  const providerSelect = byId("project-provider-select");
  providerSelect.replaceChildren(makeOption("Padrão (Codex)", ""));
  for (const provider of state.providers) {
    providerSelect.append(makeOption(
      provider.supported ? provider.name : `${provider.name} · indisponível`,
      provider.id,
      !provider.supported,
    ));
  }
  providerSelect.value = project?.provider_id || "";
  await populateProjectModelSelect(providerSelect.value || "codex", project?.model_id || "");

  byId("project-modal-backdrop").classList.remove("hidden");
  if (focusAI) {
    byId("project-ai-brief").focus();
  } else {
    const nameInput = byId("project-name-input");
    nameInput.focus();
    nameInput.select();
  }
}

async function populateProjectModelSelect(providerId, selectedModelId = "") {
  const modelSelect = byId("project-model-select");
  modelSelect.replaceChildren(makeOption("Padrão do provider", ""));
  try {
    const catalog = await loadModels(providerId || "codex");
    for (const model of catalog?.models || []) {
      modelSelect.append(makeOption(
        `${model.name || model.id}${model.default ? " · padrão" : ""}`,
        model.id,
        !model.selectable,
      ));
    }
    modelSelect.value = selectedModelId;
  } catch {
    modelSelect.value = "";
  }
}

function closeProjectModal() {
  byId("project-modal-backdrop").classList.add("hidden");
  state.editingProjectId = "";
}

async function generateProjectWithAI() {
  const brief = byId("project-ai-brief").value.trim();
  if (!brief) {
    showToast("Descreva o projeto para a IA estruturar o padrão.");
    byId("project-ai-brief").focus();
    return;
  }
  const btn = byId("project-ai-generate");
  const originalText = btn.textContent;
  btn.disabled = true;
  btn.textContent = "✦ Estruturando…";
  try {
    const selection = currentSelection();
    const draft = await api("/projects/ai-draft", {
      method: "POST",
      body: JSON.stringify({
        brief,
        provider_id: selection.provider_id || "codex",
        model_id: selection.model_id || "",
        reasoning_effort: selection.reasoning_effort || "",
      }),
    });
    if (draft.name) byId("project-name-input").value = draft.name;
    if (draft.workspace) byId("project-workspace-input").value = draft.workspace;
    if (draft.repositories?.length) byId("project-repos-input").value = draft.repositories.join("\n");
    if (draft.description) byId("project-description-input").value = draft.description;
    if (draft.instructions) byId("project-instructions-input").value = draft.instructions;
    showToast("Estrutura gerada pela IA. Revise e clique em Salvar projeto.");
  } catch (error) {
    showToast(error.message);
  } finally {
    btn.disabled = false;
    btn.textContent = originalText;
  }
}

async function saveProjectForm(event) {
  event.preventDefault();
  const payload = {
    name: byId("project-name-input").value.trim() || "Novo projeto",
    workspace: byId("project-workspace-input").value.trim(),
    repositories: byId("project-repos-input").value.split("\n").map((line) => line.trim()).filter(Boolean),
    description: byId("project-description-input").value.trim(),
    instructions: byId("project-instructions-input").value.trim(),
    routing_mode: byId("project-routing-select").value,
    provider_id: byId("project-provider-select").value,
    model_id: byId("project-model-select").value,
    approval_policy: byId("project-approval-select").value,
  };
  try {
    let saved;
    if (state.editingProjectId) {
      saved = await api(`/projects/${encodeURIComponent(state.editingProjectId)}`, {
        method: "PATCH",
        body: JSON.stringify(payload),
      });
    } else {
      saved = await api("/projects", {
        method: "POST",
        body: JSON.stringify(payload),
      });
    }
    state.selectedProjectIds.add(saved.id);
    state.activeProjectId = saved.id;
    closeProjectModal();
    await loadProjects();
    renderConversationList();
    updateTopbarProjectState();
    showToast("Projeto salvo.");
  } catch (error) {
    showToast(error.message);
  }
}

async function deleteActiveProject() {
  if (!state.editingProjectId) return;
  const projectID = state.editingProjectId;
  try {
    await api(`/projects/${encodeURIComponent(projectID)}`, { method: "DELETE" });
    state.selectedProjectIds.delete(projectID);
    if (state.activeProjectId === projectID) {
      state.activeProjectId = Array.from(state.selectedProjectIds).pop() || "";
    }
    if (state.conversation?.project_id === projectID) state.conversation.project_id = "";
    closeProjectModal();
    await Promise.all([loadProjects(), loadConversations()]);
    updateTopbarProjectState();
    showToast("Projeto removido.");
  } catch (error) {
    showToast(error.message);
  }
}

async function assignCurrentConversationProject(projectID) {
  if (!state.conversation) return;
  try {
    state.conversation = await api(`/conversations/${encodeURIComponent(state.conversation.id)}/project`, {
      method: "PATCH",
      body: JSON.stringify({ project_id: projectID }),
    });
    await loadConversations();
    updateTopbarProjectState();
    showToast(projectID ? "Conversa vinculada ao projeto." : "Conversa desvinculada do projeto.");
  } catch (error) {
    showToast(error.message);
    updateTopbarProjectState();
  }
}

async function assignCurrentConversationParent(parentConversationID) {
  if (!state.conversation) return;
  try {
    state.conversation = await api(`/conversations/${encodeURIComponent(state.conversation.id)}/parent`, {
      method: "PATCH",
      body: JSON.stringify({ parent_conversation_id: parentConversationID }),
    });
    await loadConversations();
    updateTopbarProjectState();
    showToast(parentConversationID ? "Vinculada à conversa raiz." : "Conversa promovida a conversa raiz independente.");
  } catch (error) {
    showToast(error.message);
    updateTopbarProjectState();
  }
}

async function spawnSubConversation(parentID = "") {
  const targetParent =
    parentID ||
    state.conversation?.root_conversation_id ||
    state.conversation?.parent_conversation_id ||
    state.conversation?.id ||
    "";
  if (!targetParent) return;
  await createConversation({ parentConversationId: targetParent });
}

async function loadConversations() {
  const result = await api("/conversations?summary=true");
  const incoming = result.conversations || [];
  const changed = state.conversationSyncInitialized
    ? window.CockpitInteractions.changedConversations(state.conversations, incoming, state.conversation?.id)
    : [];
  const newlyUnread = changed.filter((conversation) => !state.unreadConversations.has(conversation.id));
  state.conversations = incoming;
  state.conversationSyncInitialized = true;
  for (const conversation of changed) state.unreadConversations.add(conversation.id);
  state.conversationSyncErrorShown = false;
  renderProjectList();
  renderConversationList();
  updateTopbarProjectState();
  if (newlyUnread.length) notifyConversationUpdates(newlyUnread);
  if (state.conversation && !state.activeTurn && changed.some((item) => item.id === state.conversation.id)) {
    await refreshSelectedConversation();
  }
}

function toggleArchivedConversationsView() {
  state.showArchived = !state.showArchived;
  renderProjectList();
  renderConversationList();
  updateTopbarProjectState();
}

async function toggleConversationArchive(conversationId, archived) {
  try {
    const updated = await api(`/conversations/${encodeURIComponent(conversationId)}/archive`, {
      method: "PATCH",
      body: JSON.stringify({ archived }),
    });
    if (state.conversation?.id === conversationId) {
      state.conversation.archived = updated.archived;
    }
    await loadConversations();
    showToast(archived ? "Conversa arquivada." : "Conversa desarquivada.");
  } catch (error) {
    showToast(error.message);
  }
}

function buildConversationItemElement(conversation, options = {}) {
  const { isChild = false, childCount = 0, collapsed = false, onToggleCollapse = null } = options;
  const wrapper = document.createElement("div");
  wrapper.className = `conversation-item-wrap${isChild ? " is-child" : ""}${childCount > 0 ? " is-root-with-children" : ""}`;

  const button = document.createElement("button");
  button.type = "button";
  button.className = `conversation-link${conversation.id === state.conversation?.id ? " selected" : ""}${isChild ? " conversation-child-link" : ""}`;

  const headerRow = document.createElement("div");
  headerRow.className = "conversation-title-row";
  if (isChild) {
    const branchIcon = document.createElement("span");
    branchIcon.className = "conversation-branch-icon";
    branchIcon.textContent = "↳";
    branchIcon.title = "Sub-sessão vinculada à conversa raiz";
    headerRow.append(branchIcon);
  } else if (childCount > 0) {
    const rootIcon = document.createElement("span");
    rootIcon.className = "conversation-root-icon";
    rootIcon.textContent = "⑂";
    rootIcon.title = "Conversa raiz";
    headerRow.append(rootIcon);
  }
  const title = document.createElement("span");
  title.className = "conversation-title";
  title.textContent = conversation.title || "Nova conversa";
  headerRow.append(title);
  button.append(headerRow);

  const badgesRow = document.createElement("div");
  badgesRow.className = "conversation-badges-row";
  if (childCount > 0) {
    const groupBadge = document.createElement("span");
    groupBadge.className = "conversation-group-badge";
    groupBadge.textContent = `${collapsed ? "▸" : "▾"} ${childCount} sub-sessão${childCount > 1 ? "ões" : ""}`;
    groupBadge.title = collapsed ? "Expandir sub-sessões" : "Recolher sub-sessões";
    if (onToggleCollapse) {
      groupBadge.addEventListener("click", (event) => {
        event.stopPropagation();
        onToggleCollapse();
      });
    }
    badgesRow.append(groupBadge);
  }
  if (conversation.latest_turn_status === "running") {
    const running = document.createElement("span");
    running.className = "conversation-state conversation-state-running";
    running.textContent = "Em execução";
    badgesRow.append(running);
  }
  if (state.unreadConversations.has(conversation.id)) {
    const unread = document.createElement("span");
    unread.className = "conversation-state conversation-state-unread";
    unread.textContent = "Atualizada";
    badgesRow.append(unread);
  }
  if (state.selectedProjectIds.size !== 1 && conversation.project_id && !isChild) {
    const proj = state.projects.find((item) => item.id === conversation.project_id);
    if (proj) {
      const tag = document.createElement("span");
      tag.className = "conversation-project-tag";
      tag.textContent = `📁 ${proj.name}`;
      badgesRow.append(tag);
    }
  }
  if (badgesRow.childNodes.length > 0) {
    button.append(badgesRow);
  }

  const date = document.createElement("small");
  date.textContent = formatDate(conversation.updated_at);
  button.append(date);
  button.addEventListener("click", () => openConversation(conversation.id));

  wrapper.append(button);

  const actions = document.createElement("div");
  actions.className = "conversation-item-actions";
  if (!conversation.assistant_mode) {
    const spawnQuickBtn = document.createElement("button");
    spawnQuickBtn.type = "button";
    spawnQuickBtn.className = "conversation-spawn-btn";
    spawnQuickBtn.textContent = "↳";
    spawnQuickBtn.title = "Criar sub-sessão vinculada a esta conversa raiz";
    spawnQuickBtn.setAttribute("aria-label", "Criar sub-sessão");
    spawnQuickBtn.addEventListener("click", (event) => {
      event.stopPropagation();
      const rootID = conversation.root_conversation_id || conversation.parent_conversation_id || conversation.id;
      spawnSubConversation(rootID);
    });
    actions.append(spawnQuickBtn);
  }

  const archiveBtn = document.createElement("button");
  archiveBtn.type = "button";
  archiveBtn.className = "conversation-archive-btn";
  archiveBtn.textContent = conversation.archived ? "↩" : "📥";
  archiveBtn.title = conversation.archived ? "Desarquivar conversa" : "Arquivar conversa";
  archiveBtn.setAttribute("aria-label", `${archiveBtn.title} ${conversation.title || ""}`);
  archiveBtn.addEventListener("click", (event) => {
    event.stopPropagation();
    toggleConversationArchive(conversation.id, !conversation.archived);
  });
  actions.append(archiveBtn);
  wrapper.append(actions);
  return wrapper;
}

function renderConversationList() {
  const list = byId("conversation-list");
  list.replaceChildren();
  const visible = window.CockpitInteractions?.filterConversations
    ? window.CockpitInteractions.filterConversations(
        state.conversations,
        state.selectedProjectIds,
        state.showArchived,
      )
    : state.conversations.filter((item) => Boolean(item.archived) === Boolean(state.showArchived));

  if (!visible.length) {
    const empty = document.createElement("div");
    empty.className = "project-empty";
    empty.textContent = state.showArchived ? "Nenhuma conversa arquivada" : "Nenhuma conversa encontrada";
    list.append(empty);
    return;
  }

  const groups = window.CockpitInteractions?.groupConversationsByRoot
    ? window.CockpitInteractions.groupConversationsByRoot(visible)
    : visible.map((item) => ({ rootId: item.id, root: item, children: [] }));

  for (const group of groups) {
    const hasChildren = group.children.length > 0;
    if (!hasChildren && group.root) {
      list.append(buildConversationItemElement(group.root));
      continue;
    }

    const groupEl = document.createElement("div");
    groupEl.className = "conversation-group";
    const isCollapsed = state.collapsedRootIds.has(group.rootId);
    const toggleCollapse = () => {
      if (state.collapsedRootIds.has(group.rootId)) {
        state.collapsedRootIds.delete(group.rootId);
      } else {
        state.collapsedRootIds.add(group.rootId);
      }
      renderConversationList();
    };

    if (group.root) {
      groupEl.append(
        buildConversationItemElement(group.root, {
          childCount: group.children.length,
          collapsed: isCollapsed,
          onToggleCollapse: toggleCollapse,
        }),
      );
    } else {
      const fullRoot = state.conversations.find((item) => item.id === group.rootId);
      const headerBtn = document.createElement("button");
      headerBtn.type = "button";
      headerBtn.className = "conversation-orphan-root";
      headerBtn.textContent = `⑂ Raiz: ${fullRoot?.title || group.rootId.slice(0, 12)} (${group.children.length})`;
      headerBtn.addEventListener("click", () => openConversation(group.rootId));
      groupEl.append(headerBtn);
    }

    if (!isCollapsed) {
      const childrenContainer = document.createElement("div");
      childrenContainer.className = "conversation-children";
      for (const child of group.children) {
        childrenContainer.append(buildConversationItemElement(child, { isChild: true }));
      }
      groupEl.append(childrenContainer);
    }
    list.append(groupEl);
  }
}

function notifyConversationUpdates(conversations) {
  const names = conversations.map((item) => item.title || "Conversa").slice(0, 3);
  const suffix = conversations.length > names.length ? ` e mais ${conversations.length - names.length}` : "";
  showToast(`${names.join(", ")}${suffix}: atualização recebida.`);
  if (typeof Notification !== "undefined" && Notification.permission === "granted" && document.hidden) {
    for (const conversation of conversations) {
      new Notification("Conversa atualizada", {
        body: conversation.title || "Uma conversa do Cockpit recebeu uma atualização.",
        tag: `cockpit-conversation-${conversation.id}`,
      });
    }
  }
}

async function refreshSelectedConversation() {
  const conversationID = state.conversation?.id;
  if (!conversationID) return;
  const conversation = await api(`/conversations/${encodeURIComponent(conversationID)}`);
  if (state.conversation?.id !== conversationID) return;
  state.conversation = conversation;
  renderConversation();
  renderConversationList();
}

async function requestConversationNotifications() {
  if (typeof Notification === "undefined") {
    showToast("Notificações do sistema não estão disponíveis neste ambiente.");
    return;
  }
  if (Notification.permission === "denied") {
    showToast("Notificações bloqueadas nas permissões do sistema.");
    return;
  }
  try {
    if (Notification.permission !== "granted") {
      const permission = await Notification.requestPermission();
      if (permission !== "granted") {
        showToast("As notificações do sistema não foram autorizadas.");
        return;
      }
    }
  } catch (error) {
    showToast(`Não foi possível ativar notificações: ${error.message}`);
    return;
  }
  showToast("Notificações do sistema ativadas para conversas atualizadas.");
  updateNotificationButton();
}

function updateNotificationButton() {
  const button = byId("notification-toggle");
  if (!button) return;
  const supported = typeof Notification !== "undefined";
  button.classList.toggle("hidden", !supported);
  if (!supported) return;
  const enabled = Notification.permission === "granted";
  button.textContent = "🔔";
  button.classList.toggle("notification-enabled", enabled);
  button.title = enabled ? "Notificações do sistema ativadas" : "Ativar notificações de conversas";
  button.setAttribute("aria-label", button.title);
}

function checkConversationUpdates() {
  if (state.conversationSyncInProgress || !state.conversationSyncInitialized) return;
  state.conversationSyncInProgress = true;
  loadConversations()
    .catch((error) => {
      if (state.conversationSyncErrorShown) return;
      state.conversationSyncErrorShown = true;
      showToast(`Não foi possível verificar atualizações: ${error.message}`);
    })
    .finally(() => { state.conversationSyncInProgress = false; });
}

async function createConversation(options = {}) {
  const parentConversationId = typeof options === "object" && options !== null && !(options instanceof Event)
    ? options.parentConversationId || ""
    : "";
  try {
    const parentConv = parentConversationId
      ? state.conversations.find((item) => item.id === parentConversationId)
      : null;
    const effectiveProjectId = parentConv?.project_id || state.activeProjectId || "";
    const activeProject = state.projects.find((item) => item.id === effectiveProjectId);
    if (activeProject && !state.conversation) {
      applyProjectDefaultsToDraft(activeProject);
    }
    const selection = currentSelection();
    const providerId = activeProject?.provider_id || selection.provider_id || "codex";
    if (!selection.model_id) await loadModels(providerId);
    const initial = currentSelection();
    const conversation = await api("/conversations", {
      method: "POST",
      body: JSON.stringify({
        project_id: effectiveProjectId,
        parent_conversation_id: parentConversationId,
        provider_id: activeProject?.provider_id || initial.provider_id || "codex",
        model_id: activeProject?.model_id || initial.model_id,
        reasoning_effort: activeProject?.reasoning_effort || initial.reasoning_effort || "",
        approval_policy: activeProject?.approval_policy || initial.approval_policy || "on-request",
        routing_mode: activeProject?.routing_mode || initial.routing_mode || "manual",
      }),
    });
    if (parentConversationId) {
      const rootID = conversation.root_conversation_id || parentConversationId;
      state.collapsedRootIds.delete(rootID);
    }
    state.conversation = conversation;
    await loadConversations();
    renderConversation();
    updateSelectionControls();
    updateTopbarProjectState();
    byId("message-input").focus();
    if (parentConversationId) {
      showToast("Sub-sessão criada e agrupada à conversa raiz.");
    }
    return true;
  } catch (error) {
    showToast(error.message);
    return false;
  }
}

async function createAssistantSession() {
  try {
    const result = await api("/assistant/providers");
    const provider = result.providers.find((item) => item.id === "codex" && item.supported);
    if (!provider) throw new Error("Instale e autentique um provider com ferramentas estruturadas para usar o Assistente Cockpit.");
    const catalog = await loadModels(provider.id);
    const model = catalog.models.find((item) => item.default && item.selectable)
      || catalog.models.find((item) => item.selectable);
    if (!model) throw new Error("O Codex não disponibilizou um modelo selecionável.");
    const conversation = await api("/assistant/sessions", {
      method: "POST",
      body: JSON.stringify({ model_id: model.id }),
    });
    state.activeProjectId = "";
    state.conversation = conversation;
    await loadConversations();
    renderConversation();
    updateSelectionControls();
    updateTopbarProjectState();
    byId("message-input").focus();
  } catch (error) {
    showToast(error.message);
  }
}

async function openConversation(id, resumeStream = true) {
  setConversationLoading(true);
  try {
    const previousConversationID = state.conversation?.id;
    state.conversation = await api(`/conversations/${encodeURIComponent(id)}`);
    state.unreadConversations.delete(id);
    if ( previousConversationID && previousConversationID !== id && state.attachments.length ) {
      state.attachments = [];
      renderAttachments();
      showToast("Anexos do rascunho removidos ao trocar de conversa.");
    }
    const providerId = state.conversation.selection.provider_id;
    const provider = state.providers.find((item) => item.id === providerId && item.supported);
    if (provider) {
      try { await loadModels(providerId); } catch (error) { showToast(error.message); }
    }
    updateSelectionControls();
    renderConversation();
    renderProjectList();
    renderConversationList();
    updateTopbarProjectState();
    updateAttachmentControls();
    const running = state.conversation.turns.find((turn) => turn.status === "running");
    if (running && resumeStream) streamTurn(running.id);
  } catch (error) {
    showToast(error.message);
  } finally {
    setConversationLoading(false);
  }
}

function setConversationLoading(loading, label = "Carregando conversa…") {
  const indicator = byId("conversation-loading");
  if (!indicator) return;
  indicator.classList.toggle("hidden", !loading);
  byId("conversation-loading-label").textContent = label;
}

function renderConversation() {
  updateAttachmentControls();
  const welcome = byId("welcome");
  const list = byId("message-list");
  if (!state.conversation) {
    welcome.classList.remove("hidden");
    list.classList.add("hidden");
    list.replaceChildren();
    updateSendButton();
    byId("cancel-turn").classList.add("hidden");
    return;
  }
  welcome.classList.add("hidden");
  list.classList.remove("hidden");
  list.replaceChildren();
  let runningTurnID = null;
  for (const turn of state.conversation.turns) {
    const userMessage = state.conversation.messages.find((message) => message.id === turn.user_message_id);
    const assistantMessage = state.conversation.messages.find((message) => message.id === turn.assistant_message_id);
    if (userMessage) list.append(messageBubble("user", userMessage.content, userMessage.attachments));
    const assistant = messageBubble("assistant", assistantMessage?.content || "");
    assistant.dataset.turnId = turn.id;
    if (turn.status === "running") {
      runningTurnID = turn.id;
      const pending = document.createElement("div");
      pending.className = "typing-indicator";
      pending.innerHTML = "<i></i><i></i><i></i><span>Executando</span>";
      assistant.append(pending);
    }
    if (state.conversation.assistant_mode) {
      for (const action of state.conversation.assistant_actions || []) {
        if (action.turn_id === turn.id) assistant.append(renderAssistantAction(action));
      }
      for (const entry of state.conversation.assistant_audit || []) {
        if (entry.turn_id === turn.id) assistant.append(renderAssistantAuditEvent(entry));
      }
    }
    list.append(assistant, turnDetails(turn));
  }
  setActiveTurn(runningTurnID, state.conversation.id);
  scrollToBottom();
}

function scrollToBottom() {
  const view = document.querySelector(".conversation-view");
  if (view) {
    requestAnimationFrame(() => {
      view.scrollTop = view.scrollHeight;
    });
  }
}

function messageBubble(role, content, attachments = []) {
  const row = document.createElement("article");
  row.className = `message-row ${role}`;
  const avatar = document.createElement("div");
  avatar.className = role === "user" ? "avatar user-avatar" : "avatar agent-avatar";
  avatar.textContent = role === "user" ? "L" : "C";
  const body = document.createElement("div");
  body.className = "message-body";
  const label = document.createElement("div");
  label.className = "message-author";
  label.textContent = role === "user" ? "Você" : "Cockpit Agent";
  body.append(label);
  if (content) {
    const text = document.createElement("div");
    text.className = "message-content";
    if (role === "assistant") {
      text.dataset.markdown = content;
      renderMarkdown(text, content);
    } else {
      text.classList.add("plain-text");
      text.textContent = content;
    }
    body.append(text);
  }
  if (attachments?.length) {
    const chips = document.createElement("div");
    chips.className = "message-attachments";
    for (const attachment of attachments) {
      const chip = document.createElement("div");
      chip.className = "message-attachment";
      if (isImageAttachment(attachment)) {
        const preview = document.createElement("img");
        preview.className = "message-image-preview";
        preview.src = imageDataURL(attachment);
        preview.alt = attachment.name;
        preview.loading = "lazy";
        chip.append(preview);
      }
      const name = document.createElement("span");
      name.textContent = `${isImageAttachment(attachment) ? "▧" : "▤"} ${attachment.name}`;
      chip.append(name);
      chips.append(chip);
    }
    body.append(chips);
  }
  row.append(avatar, body);
  return row;
}

function renderMarkdown(element, content) {
  const renderer = window.CockpitMarkdown?.render;
  if (typeof renderer !== "function") {
    element.textContent = content;
    if (!state.markdownWarningShown) {
      state.markdownWarningShown = true;
      showToast("O renderizador Markdown não carregou. Reabra o app para atualizar os arquivos.");
    }
    return;
  }
  element.innerHTML = renderer(content);
}

function turnDetails(turn) {
  const details = document.createElement("details");
  details.className = "turn-details";
  const summary = document.createElement("summary");
  const status = document.createElement("span");
  status.className = `status-pill status-${turn.status}${turn.routing_mode === "automatic" ? " status-auto" : ""}`;
  status.textContent = statusLabel(turn.status);
  const meta = document.createElement("span");
  meta.textContent = `${turn.provider_id} · ${turn.model_id}${turn.task_class ? ` · ${taskClassLabel(turn.task_class)}` : ""}`;
  summary.append(status, meta);
  const grid = document.createElement("div");
  grid.className = "metadata-grid";
  const fields = [
    ["Roteamento", turn.routing_mode === "automatic" ? "Automático" : "Manual"],
    ["Classe", taskClassLabel(turn.task_class) || "não classificado"],
    ["Motivo da rota", turn.route_reason || "não informado"],
    ["Esforço", turn.reasoning_effort || "padrão do modelo"],
    ["Catálogo", turn.catalog_source || "indisponível"],
    ["Descoberto", turn.catalog_discovered_at ? formatDate(turn.catalog_discovered_at) : "indisponível"],
    ["CLI", turn.cli_version || "indisponível"],
    ["Runtime", turn.runtime_version || "indisponível"],
    ["Adapter", turn.adapter_version || "indisponível"],
    ["Sessão nativa", turn.native_session_id || "indisponível"],
    ["Turno nativo", turn.native_turn_id || "indisponível"],
    ["Contexto", turn.context_transfer_mode || "indisponível"],
    ["Sandbox", turn.sandbox || "indisponível"],
    ["Aprovações", turn.approval_policy || "indisponível"],
    ["Workspace", turn.workspace || "indisponível"],
    ["Início", formatDate(turn.started_at) || "indisponível"],
    ["Conclusão", turn.completed_at ? formatDate(turn.completed_at) : "em andamento"],
    ["Duração", turn.duration_ms ? `${turn.duration_ms} ms` : "indisponível"],
    ["Exit code", turn.exit_code ?? "indisponível"],
    ["Mensagens no contexto", turn.context_source_message_ids?.length ?? 0],
    ["Uso reportado", formatUsage(turn.usage)],
  ];
  for (const [label, value] of fields) {
    const cell = document.createElement("div");
    const key = document.createElement("small");
    key.textContent = label;
    const content = document.createElement("span");
    content.textContent = value;
    cell.append(key, content);
    grid.append(cell);
  }
  details.append(summary, grid);
  const eventsHeading = document.createElement("div");
  eventsHeading.className = "events-heading";
  eventsHeading.textContent = "Eventos de execução";
  details.append(eventsHeading);
  if (turn.events?.length) {
    const eventList = document.createElement("ul");
    eventList.className = "execution-events";
    for (const event of turn.events) {
      const item = document.createElement("li");
      const label = document.createElement("span");
      label.textContent = event.type;
      const metadata = document.createElement("small");
      metadata.textContent = [event.item_id, event.decision, formatDate(event.occurred_at)].filter(Boolean).join(" · ");
      item.append(label, metadata);
      eventList.append(item);
    }
    details.append(eventList);
  } else {
    const emptyEvents = document.createElement("p");
    emptyEvents.className = "events-empty";
    emptyEvents.textContent = "O provider não informou eventos de execução para este turno.";
    details.append(emptyEvents);
  }
  if (turn.error) {
    const error = document.createElement("p");
    error.className = "turn-error";
    error.textContent = turn.error;
    details.append(error);
  }
  return details;
}

async function sendMessage() {
  if (state.activeTurn || state.sendingMessage) return;
  if (!state.conversation && !(await createConversation())) return;
  const input = byId("message-input");
  const content = input.value.trim();
  const attachments = [...state.attachments];
  if (!content && !attachments.length) return;
  if (attachments.some(isImageAttachment) && currentSelection().provider_id !== "codex") {
    showToast("Este turno com imagem será encaminhado ao Codex para análise.");
  }
  state.sendingMessage = true;
  updateSendButton();
  try {
    const messagePath = state.conversation.assistant_mode
      ? `/assistant/sessions/${state.conversation.id}/messages`
      : `/conversations/${state.conversation.id}/messages`;
    const result = await api(messagePath, {
      method: "POST",
      body: JSON.stringify({ content, attachments }),
    });
    input.value = "";
    state.attachments = [];
    renderAttachments();
    resizeInput();
    closePickers();
    byId("routing-panel").classList.add("hidden");
    const conversationID = state.conversation.id;
    setActiveTurn(result.turn_id, conversationID);
    await openConversation(conversationID, false);
    await streamTurn(result.turn_id, conversationID, Boolean(state.conversation?.assistant_mode));
  } catch (error) {
    showToast(error.message);
  } finally {
    state.sendingMessage = false;
    updateSendButton();
  }
}

async function streamTurn(turnId, conversationID = state.conversation?.id, assistantMode = Boolean(state.conversation?.assistant_mode)) {
  if (state.activeTurn && state.activeTurn !== turnId) return;
  if (!conversationID) return;
  setActiveTurn(turnId, conversationID);
  let finished = false;
  while (!finished && state.activeTurn === turnId) {
    try {
      const after = state.eventIDs.get(turnId) || 0;
      const eventPath = assistantMode
        ? `/api/v1/assistant/sessions/${conversationID}/turns/${turnId}/events?after=${after}`
        : `/api/v1/conversations/${conversationID}/turns/${turnId}/events?after=${after}`;
      let response = await fetch(eventPath, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (response.status === 401 && (await refreshToken())) {
        response = await fetch(eventPath, {
          headers: { Authorization: `Bearer ${token}` },
        });
      }
      if (!response.ok || !response.body) throw new Error("A transmissão foi interrompida.");
      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      while (state.activeTurn === turnId) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        let boundary;
        while ((boundary = buffer.indexOf("\n\n")) >= 0) {
          const raw = buffer.slice(0, boundary);
          buffer = buffer.slice(boundary + 2);
          if (handleSSE(raw, turnId)) finished = true;
        }
        if (finished) break;
      }
      if (finished) {
        await refreshAfterTurn(conversationID);
        break;
      }
    } catch (error) {
      showToast(error.message);
    }
    if (!finished && state.activeTurn === turnId) await new Promise((resolve) => setTimeout(resolve, 1200));
  }
}

function handleSSE(raw, turnId) {
  let eventType = "message";
  let eventID = 0;
  let data = {};
  for (const line of raw.split("\n")) {
    if (line.startsWith("id: ")) eventID = Number(line.slice(4));
    if (line.startsWith("event: ")) eventType = line.slice(7);
    if (line.startsWith("data: ")) {
      try { data = JSON.parse(line.slice(6)); } catch { data = {}; }
    }
  }
  if (eventID) state.eventIDs.set(turnId, eventID);
  if (eventType === "delta") appendDelta(turnId, data.content || "");
  if (eventType === "status") updateTurnStatus(turnId, data.status || "");
  if (eventType === "approval_required") showApproval(turnId, data);
  if (eventType === "approval_resolved") resolveApproval(turnId, data);
  if (eventType === "assistant_tool_started") showAssistantToolActivity(turnId, data);
  if (eventType === "assistant_tool_completed") completeAssistantToolActivity(data);
  if (eventType === "assistant_action_required") showAssistantAction(turnId, data);
  if (eventType === "assistant_action_decided" || eventType === "assistant_action_resolved") {
    resolveAssistantAction(data.action_id, data.status || data.decision);
  }
  return eventType === "complete" || eventType === "error";
}

function showAssistantToolActivity(turnId, data) {
  const row = document.querySelector(`[data-turn-id="${CSS.escape(turnId)}"]`);
  const body = row?.querySelector(".message-body");
  if (!body || body.querySelector(`[data-assistant-tool-id="${CSS.escape(data.activity_id)}"]`)) return;
  const activity = document.createElement("div");
  activity.className = "assistant-tool-activity";
  activity.dataset.assistantToolId = data.activity_id;
  activity.textContent = `${assistantToolLabel(data.tool)} · consultando`;
  body.append(activity);
}

function completeAssistantToolActivity(data) {
  const activity = document.querySelector(`[data-assistant-tool-id="${CSS.escape(data.activity_id)}"]`);
  if (!activity) return;
  activity.textContent = `${assistantToolLabel(data.tool)} · ${data.status === "completed" ? "concluído" : "falhou"}`;
  activity.classList.toggle("failed", data.status !== "completed");
}

function assistantToolLabel(tool) {
  return ({
    list_conversations: "Consultando conversas",
    read_conversation: "Lendo conversa",
    propose_send_message: "Preparando envio",
    spawn_conversation: "Criando sub-sessão agrupada",
  })[tool] || `Ferramenta ${tool || "desconhecida"}`;
}

function renderAssistantAuditEvent(entry) {
  const activity = document.createElement("div");
  activity.className = `assistant-tool-activity${entry.status === "failed" ? " failed" : ""}`;
  const details = [
    assistantToolLabel(entry.tool),
    assistantAuditStatus(entry.status),
    entry.conversation_id && `chat ${entry.conversation_id}`,
    entry.message_ids?.length && `${entry.message_ids.length} mensagem(ns) consultada(s)`,
    entry.destination_turn_id && `turno ${entry.destination_turn_id}`,
    entry.duration_ms != null && `${entry.duration_ms} ms`,
    entry.error,
  ].filter(Boolean);
  activity.textContent = details.join(" · ");
  return activity;
}

function assistantAuditStatus(status) {
  return ({
    completed: "concluído",
    awaiting_approval: "aguardando aprovação",
    approved: "aprovado",
    rejected: "recusado",
    queued: "enviado",
    expired: "expirado",
    invalidated: "invalidado",
    cancelled: "cancelado",
    interrupted: "interrompido",
    failed: "falhou",
  })[status] || status || "";
}

function showAssistantAction(turnId, data) {
  const action = {
    id: data.action_id,
    turn_id: turnId,
    conversation_id: data.conversation_id,
    destination_selection: data.destination_selection || {},
    content: data.content,
    status: "pending",
    expires_at: data.expires_at,
  };
  state.conversation.assistant_actions ||= [];
  if (!state.conversation.assistant_actions.some((item) => item.id === action.id)) {
    state.conversation.assistant_actions.push(action);
  }
  const row = document.querySelector(`[data-turn-id="${CSS.escape(turnId)}"]`);
  if (row && !row.querySelector(`[data-assistant-action-id="${CSS.escape(action.id)}"]`)) {
    row.querySelector(".message-body").append(renderAssistantAction(action));
  }
  scrollToBottom();
}

function renderAssistantAction(action) {
  const card = document.createElement("section");
  card.className = "assistant-action-card";
  card.dataset.assistantActionId = action.id;
  const title = document.createElement("strong");
  const target = state.conversations.find((item) => item.id === action.conversation_id);
  title.textContent = `Enviar mensagem para ${target?.title || action.conversation_id}`;
  const selection = document.createElement("small");
  const chosen = action.destination_selection || {};
  selection.textContent = `Destino: ${chosen.provider_id || "provider"} · ${chosen.model_id || "modelo"}${action.expires_at ? ` · expira ${formatDate(action.expires_at)}` : ""}`;
  const content = document.createElement("pre");
  content.textContent = action.content || "";
  const status = document.createElement("span");
  status.className = "assistant-action-status";
  status.textContent = assistantActionStatus(action.status);
  card.append(title, selection, content, status);
  if (action.status === "queued") {
    const open = document.createElement("button");
    open.type = "button";
    open.className = "assistant-action-open";
    open.textContent = "Abrir conversa de destino";
    open.addEventListener("click", () => openConversation(action.conversation_id));
    card.append(open);
  }
  if (action.status === "pending") {
    const buttons = document.createElement("div");
    buttons.className = "assistant-action-buttons";
    for (const [decision, label] of [["reject", "Recusar"], ["approve", "Aprovar envio"]]) {
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = label;
      button.disabled = Boolean(state.activeTurn && state.activeTurn !== action.turn_id);
      button.addEventListener("click", () => decideAssistantAction(action, decision, card));
      buttons.append(button);
    }
    card.append(buttons);
  }
  return card;
}

function assistantActionStatus(status) {
  return ({
    pending: "Aguardando sua decisão",
    approved: "Aprovado — enviando",
    rejected: "Recusado — nada foi enviado",
    queued: "Turno iniciado no chat de destino",
    expired: "Proposta expirada",
    interrupted: "Sessão terminou antes de concluir a ação",
    cancelled: "Ação cancelada junto com o turno do assistente",
    invalidated: "Seleção alterada — crie uma nova proposta",
    failed: "Falha ao iniciar o turno",
  })[status] || status || "";
}

async function decideAssistantAction(action, decision, card) {
  const buttons = card.querySelectorAll("button");
  buttons.forEach((button) => { button.disabled = true; });
  try {
    await api(`/assistant/sessions/${encodeURIComponent(state.conversation.id)}/actions/${encodeURIComponent(action.id)}/decision`, {
      method: "POST",
      body: JSON.stringify({ decision }),
    });
    action.status = decision === "approve" ? "approved" : "rejected";
    resolveAssistantAction(action.id, action.status);
  } catch (error) {
    buttons.forEach((button) => { button.disabled = false; });
    showToast(error.message);
  }
}

function resolveAssistantAction(actionID, status) {
  const action = state.conversation?.assistant_actions?.find((item) => item.id === actionID);
  if (action && status) action.status = status;
  const card = document.querySelector(`[data-assistant-action-id="${CSS.escape(actionID)}"]`);
  if (!card || !status) return;
  card.querySelector(".assistant-action-status").textContent = assistantActionStatus(status);
  card.querySelectorAll(".assistant-action-buttons button").forEach((button) => { button.disabled = true; });
  if (status === "queued" && !state.activeTurn && !card.querySelector(".assistant-action-open")) {
    const open = document.createElement("button");
    open.type = "button";
    open.className = "assistant-action-open";
    open.textContent = "Abrir conversa de destino";
    open.addEventListener("click", () => openConversation(action?.conversation_id || ""));
    card.append(open);
  }
}

function updateTurnStatus(turnId, statusText) {
  if (!statusText) return;
  const row = document.querySelector(`[data-turn-id="${CSS.escape(turnId)}"]`);
  const label = row?.querySelector(".typing-indicator span");
  if (label) {
    label.textContent = statusText;
    scrollToBottom();
  }
}

function appendDelta(turnId, content) {
  const row = document.querySelector(`[data-turn-id="${CSS.escape(turnId)}"]`);
  if (!row) return;
  row.querySelector(".typing-indicator")?.remove();
  let target = row.querySelector(".message-content");
  if (!target) {
    target = document.createElement("div");
    target.className = "message-content";
    row.querySelector(".message-body").append(target);
  }
  target.dataset.markdown = `${target.dataset.markdown || ""}${content}`;
  renderMarkdown(target, target.dataset.markdown);
  scrollToBottom();
}

function showApproval(turnId, data) {
  const row = document.querySelector(`[data-turn-id="${CSS.escape(turnId)}"]`);
  if (!row || row.querySelector(`[data-approval-id="${CSS.escape(data.request_id)}"]`)) return;
  const card = document.createElement("section");
  card.className = "approval-card";
  card.dataset.approvalId = data.request_id;
  const title = document.createElement("strong");
  title.textContent = data.method === "item/commandExecution/requestApproval" ? "Aprovar execução de comando" : "Aprovar alteração de arquivo";
  card.append(title);
  const params = data.params || {};
  if (params.command) {
    const command = document.createElement("pre");
    command.textContent = Array.isArray(params.command) ? params.command.join(" ") : String(params.command);
    card.append(command);
  }
  if (params.reason || params.cwd) {
    const context = document.createElement("p");
    context.textContent = [params.reason, params.cwd && `Diretório: ${params.cwd}`].filter(Boolean).join(" · ");
    card.append(context);
  }
  const actions = document.createElement("div");
  actions.className = "approval-actions";
  const decline = document.createElement("button");
  decline.className = "approval-deny";
  decline.textContent = "Recusar";
  decline.addEventListener("click", () => decideApproval(turnId, data.request_id, "decline", card));
  const accept = document.createElement("button");
  accept.className = "approval-accept";
  accept.textContent = "Aprovar";
  const decisions = params.availableDecisions;
  const canAccept = data.method === "item/fileChange/requestApproval" || (Array.isArray(decisions) && decisions.includes("accept"));
  accept.disabled = !canAccept;
  if (!canAccept) accept.title = "Codex não ofereceu uma opção de aprovação para este pedido.";
  accept.addEventListener("click", () => decideApproval(turnId, data.request_id, "accept", card));
  actions.append(decline, accept);
  card.append(actions);
  row.querySelector(".message-body").append(card);
}

function resolveApproval(turnId, data) {
  const row = document.querySelector(`[data-turn-id="${CSS.escape(turnId)}"]`);
  const card = row?.querySelector(`[data-approval-id="${CSS.escape(data.request_id)}"]`);
  if (!card) return;
  card.replaceChildren(document.createTextNode(data.decision === "accept" ? "Ação aprovada." : "Ação recusada."));
  card.classList.add("approval-resolved");
}

async function decideApproval(turnId, requestID, decision, card) {
  try {
    await api(`/conversations/${state.conversation.id}/turns/${turnId}/approvals/${encodeURIComponent(requestID)}`, {
      method: "POST", body: JSON.stringify({ decision }),
    });
    card.replaceChildren(document.createTextNode(decision === "accept" ? "Ação aprovada." : "Ação recusada."));
    card.classList.add("approval-resolved");
  } catch (error) { showToast(error.message); }
}

async function refreshAfterTurn(conversationID) {
  if (state.conversation?.id === conversationID) {
    state.conversation = await api(`/conversations/${conversationID}`);
    renderConversation();
  }
  setActiveTurn(null, conversationID);
  await loadConversations();
}

async function cancelTurn() {
  if (!state.conversation || !state.activeTurn || state.activeTurnConversationID !== state.conversation.id) return;
  try {
    const path = state.conversation.assistant_mode
      ? `/assistant/sessions/${state.conversation.id}/turns/${state.activeTurn}/cancel`
      : `/conversations/${state.conversation.id}/turns/${state.activeTurn}/cancel`;
    await api(path, { method: "POST", body: "{}" });
    byId("cancel-turn").textContent = "Cancelando…";
  } catch (error) { showToast(error.message); }
}

async function updateSelection(patch) {
  const next = { ...currentSelection(), ...patch };
  if (!state.conversation) {
    state.draft = next;
    updateSelectionControls();
    return;
  }
  try {
    const selection = await api(`/conversations/${state.conversation.id}/selection`, {
      method: "PATCH",
      body: JSON.stringify({
        provider_id: next.provider_id, model_id: next.model_id,
        reasoning_effort: next.reasoning_effort || "", approval_policy: next.approval_policy || "on-request",
        routing_mode: next.routing_mode || "manual",
      }),
    });
    state.conversation.selection = selection;
    updateSelectionControls();
    showToast("Seleção salva para os próximos turnos.");
  } catch (error) {
    showToast(error.message);
    updateSelectionControls();
  }
}

function updateSelectionControls() {
  const selection = currentSelection();
  const assistantMode = Boolean(state.conversation?.assistant_mode);
  const provider = state.providers.find((item) => item.id === selection.provider_id);
  const catalog = state.catalogs.get(selection.provider_id);
  const model = catalog?.models.find((item) => item.id === selection.model_id);
  byId("provider-picker-label").textContent = provider?.name || selection.provider_id || "Provider";
  byId("model-picker-label").textContent = model?.name || model?.id || "Modelo";
  byId("effort-summary").textContent = selection.reasoning_effort ? effortLabel(selection.reasoning_effort) : "";
  byId("approval-picker-label").textContent = selection.approval_policy === "never" ? "Aprovar por mim" : "Pedir aprovação";
  byId("composer-disclaimer").textContent = assistantMode
    ? "Assistente Cockpit: leitura limitada de conversas; qualquer envio a outro chat exige sua aprovação explícita."
    : selection.approval_policy === "never"
    ? "O agente pode ler e alterar arquivos dentro do workspace. O provider não vai pausar para pedir aprovação (never). Sandbox: workspace-write."
    : "O agente pode ler e alterar arquivos dentro do workspace. Ações que exigem aprovação aguardam sua decisão. Sandbox: workspace-write.";
  const automatic = selection.routing_mode === "automatic";
  byId("routing-mode-label").textContent = automatic ? "Auto" : "Manual";
  byId("routing-mode-toggle").setAttribute("aria-pressed", String(automatic));
  byId("routing-mode-toggle").classList.toggle("manual-mode", !automatic);
  byId("provider-picker-toggle").disabled = automatic || assistantMode;
  byId("model-picker-toggle").disabled = automatic || assistantMode;
  byId("reasoning-effort").disabled = automatic || assistantMode || !model?.reasoning_efforts?.length;
  const catalogCaption = catalog ? `Origem: ${catalog.source} · ${formatDate(catalog.discovered_at)}` : "Catálogo do provider";
  byId("catalog-caption").textContent = catalogCaption;
  renderProviderOptions();
  renderModelOptions();
  renderEffortOptions();
  renderApprovalOptions();
  byId("provider-picker-toggle").disabled = automatic || assistantMode;
  byId("model-picker-toggle").disabled = automatic || assistantMode;
  byId("approval-picker-toggle").disabled = assistantMode;
  byId("routing-mode-toggle").disabled = assistantMode;
  byId("routing-settings-toggle").disabled = assistantMode;
  byId("attach-files").disabled = assistantMode;
  byId("voice-input").disabled = assistantMode;
  byId("message-input").placeholder = assistantMode
    ? "Peça ao assistente para consultar ou atuar nos chats…"
    : "Mensagem para o agente…";
}

function renderProviderOptions() {
  const options = byId("provider-options");
  if (!options) return;
  options.replaceChildren();
  const selected = currentSelection();
  for (const provider of state.providers) {
    const button = document.createElement("button");
    button.className = `picker-option${provider.id === selected.provider_id ? " selected" : ""}`;
    button.type = "button";
    button.disabled = !provider.supported;
    button.title = provider.status;
    const labels = document.createElement("span");
    const name = document.createElement("strong");
    name.textContent = provider.name;
    const status = document.createElement("small");
    status.textContent = provider.supported ? "Adapter de conversa disponível" : provider.detected ? "CLI detectado · adapter pendente" : "CLI não instalado";
    labels.append(name, status);
    const check = document.createElement("span");
    check.className = "checkmark";
    check.textContent = provider.id === selected.provider_id ? "✓" : "";
    button.append(labels, check);
    button.addEventListener("click", () => chooseProvider(provider.id));
    options.append(button);
  }
}

async function chooseProvider(providerId) {
  try {
    const catalog = await loadModels(providerId);
    const selectedModel = catalog.models.find((model) => model.default && model.selectable) || catalog.models.find((model) => model.selectable);
    if (!selectedModel) throw new Error("Esse provider não tem modelos selecionáveis.");
    await updateSelection({ provider_id: providerId, model_id: selectedModel.id, reasoning_effort: "" });
    closePickers();
  } catch (error) { showToast(error.message); }
}

function renderModelOptions(filter = byId("model-search")?.value || "") {
  const options = byId("model-options");
  if (!options) return;
  options.replaceChildren();
  const selection = currentSelection();
  const catalog = state.catalogs.get(selection.provider_id);
  if (!catalog) return;
  const query = filter.trim().toLocaleLowerCase("pt-BR");
  for (const model of catalog.models) {
    if (query && !`${model.name} ${model.id} ${model.description || ""}`.toLocaleLowerCase("pt-BR").includes(query)) continue;
    const button = document.createElement("button");
    button.className = `picker-option${model.id === selection.model_id ? " selected" : ""}`;
    button.type = "button";
    button.disabled = !model.selectable || model.hidden;
    const labels = document.createElement("span");
    const name = document.createElement("strong");
    name.textContent = `${model.name || model.id}${model.default ? " · padrão" : ""}`;
    const description = document.createElement("small");
    description.textContent = model.description || (model.selectable ? model.id : "Não selecionável neste provider");
    labels.append(name, description);
    const check = document.createElement("span");
    check.className = "checkmark";
    check.textContent = model.id === selection.model_id ? "✓" : "";
    button.append(labels, check);
    button.addEventListener("click", async () => {
      await updateSelection({ model_id: model.id, reasoning_effort: "" });
      closePickers();
    });
    options.append(button);
  }
}

function renderEffortOptions() {
  const select = byId("reasoning-effort");
  const selection = currentSelection();
  const model = state.catalogs.get(selection.provider_id)?.models.find((item) => item.id === selection.model_id);
  const current = selection.reasoning_effort || "";
  select.replaceChildren(makeOption("Padrão do modelo", ""));
  for (const effort of model?.reasoning_efforts || []) select.append(makeOption(effortLabel(effort), effort));
  select.value = current;
  select.disabled = !model?.reasoning_efforts?.length || selection.routing_mode === "automatic";
}

function renderApprovalOptions() {
  const current = currentSelection().approval_policy || "on-request";
  for (const button of document.querySelectorAll("[data-approval]")) {
    const selected = button.dataset.approval === current;
    button.classList.toggle("selected", selected);
    button.querySelector(".checkmark").textContent = selected ? "✓" : "";
  }
}

function togglePicker(id) {
  const panel = byId(id);
  const opening = panel.classList.contains("hidden");
  closePickers();
  panel.classList.toggle("hidden", !opening);
  const trigger = document.querySelector(`[aria-controls="${id}"]`);
  if (trigger) trigger.setAttribute("aria-expanded", String(opening));
}

function closePickers() {
  for (const panel of document.querySelectorAll(".picker-popover")) panel.classList.add("hidden");
  for (const trigger of document.querySelectorAll("[aria-expanded]")) trigger.setAttribute("aria-expanded", "false");
}

function setActiveTurn(turnID, conversationID = state.conversation?.id || "") {
  if (turnID) {
    state.activeTurn = turnID;
    state.activeTurnConversationID = conversationID;
  } else if (!conversationID || state.activeTurnConversationID === conversationID) {
    state.activeTurn = null;
    state.activeTurnConversationID = "";
  }
  updateSendButton();
  const activeInCurrentConversation = Boolean(state.activeTurn && state.activeTurnConversationID === state.conversation?.id);
  byId("cancel-turn").classList.toggle("hidden", !activeInCurrentConversation);
  byId("cancel-turn").textContent = "■ Cancelar";
}

function updateSendButton() {
  const button = byId("send-message");
  if (!button) return;
  const loading = Boolean(state.activeTurn || state.sendingMessage);
  button.disabled = loading;
  button.classList.toggle("is-loading", loading);
  button.title = state.activeTurn
    ? state.activeTurnConversationID === state.conversation?.id
      ? "Aguardando resposta do agente"
      : "Outro chat ainda está em execução"
    : state.sendingMessage ? "Enviando mensagem…" : "Enviar mensagem";
  button.setAttribute("aria-label", button.title);
}

function updateAttachmentControls() {
  const button = byId("attach-files");
  if (!button) return;
  button.disabled = state.addingAttachments;
  button.classList.toggle("is-loading", state.addingAttachments);
  button.title = state.addingAttachments ? "Lendo anexos…" : "Anexar arquivos ou imagens";
}

function resizeInput() {
  const input = byId("message-input");
  input.style.height = "auto";
  input.style.height = `${Math.min(input.scrollHeight, 180)}px`;
}

function renderAttachments() {
  const list = byId("attachment-list");
  list.replaceChildren();
  state.attachments.forEach((attachment, index) => {
    const chip = document.createElement("div");
    chip.className = "attachment-chip";
    if (isImageAttachment(attachment)) {
      const preview = document.createElement("img");
      preview.className = "attachment-image-preview";
      preview.src = imageDataURL(attachment);
      preview.alt = "";
      chip.append(preview);
    }
    const name = document.createElement("span");
    name.textContent = `${isImageAttachment(attachment) ? "▧" : "▤"} ${attachment.name}`;
    name.title = attachment.name;
    const remove = document.createElement("button");
    remove.type = "button";
    remove.textContent = "×";
    remove.setAttribute("aria-label", `Remover ${attachment.name}`);
    remove.addEventListener("click", () => {
      state.attachments.splice(index, 1);
      renderAttachments();
    });
    chip.append(name, remove);
    list.append(chip);
  });
  updateAttachmentControls();
}

function isImageAttachment(attachment) {
  return ["image/gif", "image/jpeg", "image/png", "image/webp"].includes(attachment.media_type);
}

function imageDataURL(attachment) {
  return `data:${attachment.media_type};base64,${attachment.content}`;
}

function attachmentSize(attachment) {
  if (isImageAttachment(attachment)) return Math.floor((attachment.content.length * 3) / 4);
  return new TextEncoder().encode(attachment.content).length;
}

async function addFiles(files) {
  if (state.addingAttachments) return;
  state.addingAttachments = true;
  updateAttachmentControls();
  try {
    const allowed = new Set([".c", ".cc", ".css", ".csv", ".go", ".h", ".html", ".java", ".js", ".json", ".jsx", ".log", ".md", ".py", ".rs", ".sh", ".sql", ".text", ".toml", ".ts", ".tsx", ".txt", ".xml", ".yaml", ".yml"]);
    const pending = [...files];
    if (!pending.length) return;
    if (state.attachments.length + pending.length > 5) throw new Error("Você pode anexar até 5 arquivos.");
    let addedTextBytes = state.attachments.filter((item) => !isImageAttachment(item)).reduce((sum, item) => sum + attachmentSize(item), 0);
    let addedImageBytes = state.attachments.filter(isImageAttachment).reduce((sum, item) => sum + attachmentSize(item), 0);
    const parsed = [];
    for (const file of pending) {
      const extension = file.name.slice(file.name.lastIndexOf(".")).toLocaleLowerCase("en-US");
      const imageType = imageTypeForFile(file, extension);
      if (imageType) {
        if (file.size > 4 * 1024 * 1024) throw new Error(`${file.name} excede 4 MiB.`);
        addedImageBytes += file.size;
        if (addedImageBytes > 6 * 1024 * 1024) throw new Error("As imagens juntas excedem 6 MiB.");
        const dataURL = await readFileAsDataURL(file);
        const content = dataURL.slice(dataURL.indexOf(",") + 1);
        parsed.push({ name: file.name || `screenshot.${imageExtension(imageType)}`, media_type: imageType, content });
        continue;
      }
      if (!allowed.has(extension)) throw new Error(`Tipo de arquivo não aceito: ${file.name}`);
      if (file.size > 256 * 1024) throw new Error(`${file.name} excede 256 KiB.`);
      const content = new TextDecoder("utf-8", { fatal: true }).decode(await file.arrayBuffer());
      addedTextBytes += file.size;
      if (addedTextBytes > 512 * 1024) throw new Error("Os arquivos de texto juntos excedem 512 KiB.");
      parsed.push({ name: file.name, media_type: file.type, content });
    }
    state.attachments.push(...parsed);
    renderAttachments();
  } finally {
    state.addingAttachments = false;
    updateAttachmentControls();
  }
}

function imageTypeForFile(file, extension) {
  const supported = {
    ".gif": "image/gif",
    ".jpeg": "image/jpeg",
    ".jpg": "image/jpeg",
    ".png": "image/png",
    ".webp": "image/webp",
  };
  const expectedType = supported[extension];
  if (!expectedType) return "";
  if (file.type && file.type !== expectedType) throw new Error(`Tipo de imagem inválido: ${file.name}`);
  return expectedType;
}

function imageExtension(mediaType) {
  return ({ "image/gif": "gif", "image/jpeg": "jpg", "image/png": "png", "image/webp": "webp" })[mediaType] || "png";
}

function readFileAsDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.onerror = () => reject(new Error(`Não foi possível ler ${file.name}.`));
    reader.readAsDataURL(file);
  });
}

async function pasteClipboard(event) {
  const activeElement = document.activeElement;
  if (!event.target.closest?.(".composer-area") && !activeElement?.closest?.(".composer-area")) return;
  const imageFiles = window.CockpitInteractions.clipboardImageFiles(event.clipboardData);
  if (!imageFiles.length) return;
  event.preventDefault();
  try {
    const normalized = imageFiles.map((file, index) => {
      const extensionForType = { "image/gif": "gif", "image/jpeg": "jpg", "image/png": "png", "image/webp": "webp" };
      const typeForExtension = { ".gif": "image/gif", ".jpeg": "image/jpeg", ".jpg": "image/jpeg", ".png": "image/png", ".webp": "image/webp" };
      const extension = file.name?.slice(file.name.lastIndexOf(".")).toLowerCase();
      const mediaType = file.type || typeForExtension[extension];
      const suffix = extensionForType[mediaType];
      if (!suffix) throw new Error(`Tipo de imagem não aceito: ${file.type || "desconhecido"}`);
      if (file.name && typeForExtension[extension] === mediaType && (!file.type || file.type === mediaType)) return file;
      return new File([file], `screenshot-${Date.now()}-${index + 1}.${suffix}`, { type: mediaType });
    });
    await addFiles(normalized);
    showToast("Imagem colada no rascunho.");
  } catch (error) {
    showToast(error.message);
  }
}

function handleFileDrop(event) {
  const files = [...(event.dataTransfer?.files || [])];
  if (!files.length) return;
  event.preventDefault();
  event.stopPropagation();
  state.dropDepth = 0;
  document.querySelector(".composer-wrap").classList.remove("drop-active");
  byId("drop-overlay").classList.add("hidden");
  addFiles(files).catch((error) => showToast(error.message));
}

function hasDraggedFiles(event) {
  return Array.from(event.dataTransfer?.types || []).includes("Files");
}

async function setApprovalPolicy(policy) {
  await updateSelection({ approval_policy: policy });
  closePickers();
}

async function toggleRoutingMode() {
  const mode = currentSelection().routing_mode === "automatic" ? "manual" : "automatic";
  if (mode === "automatic" && !state.routing) {
    try { state.routing = await api("/routing"); }
    catch (error) { showToast(error.message); return; }
  }
  await updateSelection({ routing_mode: mode });
}

async function openRoutingPanel() {
  closePickers();
  const panel = byId("routing-panel");
  const opening = panel.classList.contains("hidden");
  if (!opening) { panel.classList.add("hidden"); return; }
  try {
    state.routing = await api("/routing");
    const providerIDs = [...new Set(state.routing.rules.map((rule) => rule.provider_id))];
    const supportedIDs = new Set(state.providers.filter((provider) => provider.supported).map((provider) => provider.id));
    await Promise.all(providerIDs.filter((id) => supportedIDs.has(id)).map((id) => loadModels(id)));
    await renderRoutingRules();
    panel.classList.remove("hidden");
  } catch (error) { showToast(error.message); }
}

async function renderRoutingRules() {
  const container = byId("routing-rules");
  container.replaceChildren();
  for (const descriptor of routeClasses) {
    const rule = state.routing.rules.find((item) => item.task_class === descriptor.id);
    if (!rule) continue;
    const row = document.createElement("div");
    row.className = "route-row";
    const title = document.createElement("div");
    title.className = "route-class";
    const name = document.createElement("strong");
    name.textContent = descriptor.label;
    const description = document.createElement("small");
    description.textContent = descriptor.description;
    title.append(name, description);
    const providerSelect = document.createElement("select");
    providerSelect.setAttribute("aria-label", `Provider para ${descriptor.label}`);
    for (const provider of state.providers) {
      providerSelect.append(makeOption(
        provider.supported ? provider.name : `${provider.name} · indisponível`,
        provider.id,
        !provider.supported,
      ));
    }
    if (!state.providers.some((provider) => provider.id === rule.provider_id)) {
      providerSelect.append(makeOption(`${rule.provider_id} · indisponível`, rule.provider_id, true));
    }
    providerSelect.value = rule.provider_id;
    const modelSelect = document.createElement("select");
    modelSelect.setAttribute("aria-label", `Modelo para ${descriptor.label}`);
    const effortSelect = document.createElement("select");
    effortSelect.setAttribute("aria-label", `Esforço para ${descriptor.label}`);
    const fillRouteModels = async () => {
      const provider = state.providers.find((item) => item.id === providerSelect.value);
      if (!provider?.supported) {
        modelSelect.replaceChildren(makeOption(`${rule.model_id || "Modelo"} · indisponível`, rule.model_id, true));
        modelSelect.value = rule.model_id;
        effortSelect.replaceChildren(makeOption("Padrão", ""));
        effortSelect.value = "";
        effortSelect.disabled = true;
        return;
      }
      const catalog = await loadModels(providerSelect.value);
      modelSelect.replaceChildren();
      for (const model of catalog.models) {
        modelSelect.append(makeOption(`${model.name || model.id}${model.default ? " · padrão" : ""}`, model.id, !model.selectable));
      }
      const fallback = catalog.models.find((model) => model.default && model.selectable) || catalog.models.find((model) => model.selectable);
      if (!catalog.models.some((model) => model.id === rule.model_id && model.selectable)) rule.model_id = fallback?.id || "";
      modelSelect.value = rule.model_id;
      const model = catalog.models.find((item) => item.id === rule.model_id);
      effortSelect.replaceChildren(makeOption("Padrão", ""));
      for (const effort of model?.reasoning_efforts || []) effortSelect.append(makeOption(effortLabel(effort), effort));
      if (!model?.reasoning_efforts?.includes(rule.reasoning_effort)) rule.reasoning_effort = "";
      effortSelect.value = rule.reasoning_effort;
      effortSelect.disabled = !model?.reasoning_efforts?.length;
    };
    await fillRouteModels();
    providerSelect.disabled = !state.providers.some((provider) => provider.supported);
    providerSelect.addEventListener("change", async () => {
      rule.provider_id = providerSelect.value;
      rule.model_id = "";
      rule.reasoning_effort = "";
      try { await fillRouteModels(); } catch (error) { showToast(error.message); }
    });
    modelSelect.addEventListener("change", () => {
      rule.model_id = modelSelect.value;
      const model = state.catalogs.get(rule.provider_id)?.models.find((item) => item.id === rule.model_id);
      rule.reasoning_effort = model?.reasoning_efforts?.includes(rule.reasoning_effort) ? rule.reasoning_effort : "";
      effortSelect.replaceChildren(makeOption("Padrão", ""));
      for (const effort of model?.reasoning_efforts || []) effortSelect.append(makeOption(effortLabel(effort), effort));
      effortSelect.value = rule.reasoning_effort;
      effortSelect.disabled = !model?.reasoning_efforts?.length;
    });
    effortSelect.addEventListener("change", () => { rule.reasoning_effort = effortSelect.value; });
    row.append(title, providerSelect, modelSelect, effortSelect);
    container.append(row);
  }
}

async function saveRouting() {
  try {
    state.routing = await api("/routing", { method: "PUT", body: JSON.stringify(state.routing) });
    showToast("Rotas salvas. A classificação será registrada em cada turno.");
    byId("routing-panel").classList.add("hidden");
  } catch (error) { showToast(error.message); }
}

function startVoiceInput() {
  const Recognition = window.SpeechRecognition || window.webkitSpeechRecognition;
  if (!Recognition) {
    showToast("Este navegador não oferece reconhecimento de voz.");
    return;
  }
  const recognition = new Recognition();
  recognition.lang = "pt-BR";
  recognition.interimResults = false;
  recognition.onstart = () => byId("voice-input").classList.add("recording");
  recognition.onresult = (event) => {
    const transcript = event.results?.[0]?.[0]?.transcript || "";
    const input = byId("message-input");
    input.value = `${input.value}${input.value ? " " : ""}${transcript}`;
    resizeInput();
    input.focus();
  };
  recognition.onerror = (event) => showToast(`Ditado indisponível: ${event.error || "erro no microfone"}`);
  recognition.onend = () => byId("voice-input").classList.remove("recording");
  try { recognition.start(); } catch (error) { showToast(error.message); }
}

function bindUI() {
  byId("new-chat").addEventListener("click", createConversation);
  byId("assistant-chat").addEventListener("click", createAssistantSession);
  byId("welcome-new").addEventListener("click", createConversation);
  byId("create-project").addEventListener("click", createInlineProject);
  byId("ai-create-project").addEventListener("click", () => openProjectModal("", true));
  byId("clear-project-filter").addEventListener("click", () => selectProject(""));
  byId("toggle-archived-conversations").addEventListener("click", toggleArchivedConversationsView);
  byId("project-modal-close").addEventListener("click", closeProjectModal);
  byId("project-cancel-btn").addEventListener("click", closeProjectModal);
  byId("project-form").addEventListener("submit", saveProjectForm);
  byId("project-delete-btn").addEventListener("click", deleteActiveProject);
  byId("project-ai-generate").addEventListener("click", generateProjectWithAI);
  byId("project-provider-select").addEventListener("change", (event) => populateProjectModelSelect(event.target.value || "codex"));
  byId("conversation-project-select").addEventListener("change", (event) => assignCurrentConversationProject(event.target.value));
  byId("conversation-parent-select")?.addEventListener("change", (event) => assignCurrentConversationParent(event.target.value));
  byId("spawn-subchat-btn")?.addEventListener("click", () => spawnSubConversation());
  byId("project-modal-backdrop").addEventListener("click", (event) => {
    if (event.target === byId("project-modal-backdrop")) closeProjectModal();
  });
  byId("send-message").addEventListener("click", sendMessage);
  byId("cancel-turn").addEventListener("click", cancelTurn);
  byId("notification-toggle").addEventListener("click", requestConversationNotifications);
  byId("message-input").addEventListener("input", resizeInput);
  byId("message-input").addEventListener("keydown", (event) => {
    if (event.key === "Enter" && !event.shiftKey && !event.isComposing) { event.preventDefault(); sendMessage(); }
  });
  document.addEventListener("paste", pasteClipboard, true);
  byId("attach-files").addEventListener("click", () => byId("file-input").click());
  byId("file-input").addEventListener("change", async (event) => {
    try { await addFiles(event.target.files); } catch (error) { showToast(error.message); }
    event.target.value = "";
  });
  const composer = document.querySelector(".composer-wrap");
  composer.addEventListener("dragenter", (event) => {
    if (!hasDraggedFiles(event)) return;
    event.preventDefault();
    state.dropDepth += 1;
    composer.classList.add("drop-active");
    byId("drop-overlay").classList.remove("hidden");
  });
  composer.addEventListener("dragover", (event) => {
    if (!hasDraggedFiles(event)) return;
    event.preventDefault();
    if (event.dataTransfer) event.dataTransfer.dropEffect = "copy";
  });
  composer.addEventListener("dragleave", (event) => {
    if (!hasDraggedFiles(event)) return;
    state.dropDepth = Math.max(0, state.dropDepth - 1);
    if (!state.dropDepth) {
      composer.classList.remove("drop-active");
      byId("drop-overlay").classList.add("hidden");
    }
  });
  composer.addEventListener("drop", handleFileDrop);
  window.addEventListener("dragover", (event) => {
    if (hasDraggedFiles(event)) event.preventDefault();
  });
  window.addEventListener("drop", (event) => {
    if (hasDraggedFiles(event)) event.preventDefault();
  });
  byId("provider-picker-toggle").setAttribute("aria-controls", "provider-picker");
  byId("model-picker-toggle").setAttribute("aria-controls", "model-picker");
  byId("approval-picker-toggle").setAttribute("aria-controls", "approval-picker");
  byId("provider-picker-toggle").addEventListener("click", () => togglePicker("provider-picker"));
  byId("model-picker-toggle").addEventListener("click", () => togglePicker("model-picker"));
  byId("approval-picker-toggle").addEventListener("click", () => togglePicker("approval-picker"));
  byId("routing-mode-toggle").addEventListener("click", toggleRoutingMode);
  byId("routing-settings-toggle").addEventListener("click", openRoutingPanel);
  byId("routing-panel-close").addEventListener("click", () => byId("routing-panel").classList.add("hidden"));
  byId("save-routing").addEventListener("click", saveRouting);
  byId("model-search").addEventListener("input", (event) => renderModelOptions(event.target.value));
  byId("reasoning-effort").addEventListener("change", (event) => updateSelection({ reasoning_effort: event.target.value }));
  for (const button of document.querySelectorAll("[data-approval]")) {
    button.addEventListener("click", () => setApprovalPolicy(button.dataset.approval));
  }
  byId("voice-input").addEventListener("click", startVoiceInput);
  document.addEventListener("visibilitychange", () => {
    if (!document.hidden) checkConversationUpdates();
  });
  document.addEventListener("click", (event) => {
    if (!event.target.closest(".picker-anchor")) closePickers();
  });
}

async function init() {
  bindUI();
  try {
    const [workspace] = await Promise.all([api("/workspace"), loadProviders()]);
    state.defaultWorkspace = workspace.path;
    byId("workspace").textContent = workspace.path;
    await Promise.all([loadProjects(), loadConversations()]);
    const firstActive = state.conversations.find((item) => !item.archived);
    if (firstActive) await openConversation(firstActive.id);
    else updateSelectionControls();
    updateTopbarProjectState();
  } catch (error) { showToast(error.message); }
  updateNotificationButton();
  setInterval(checkConversationUpdates, 15_000);
}

function taskClassLabel(taskClass) { return routeClasses.find((item) => item.id === taskClass)?.label || taskClass || ""; }
function effortLabel(effort) { return ({ minimal: "Mínimo", low: "Baixo", medium: "Médio", high: "Alto", xhigh: "Extra alto" })[effort] || effort; }
function formatDate(value) {
  if (!value) return "";
  return new Intl.DateTimeFormat("pt-BR", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" }).format(new Date(value));
}
function statusLabel(status) { return ({ completed: "Concluído", running: "Em execução", failed: "Falhou", cancelled: "Cancelado", interrupted: "Interrompido" })[status] || status; }
function formatUsage(usage) {
  if (!usage) return "indisponível";
  const values = [];
  if (usage.input_tokens != null) values.push(`${usage.input_tokens} in`);
  if (usage.output_tokens != null) values.push(`${usage.output_tokens} out`);
  if (usage.cost != null) values.push(`${usage.cost} custo`);
  return values.join(" · ") || "indisponível";
}

init();
