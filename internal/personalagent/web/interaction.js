(function exposeInteractions(global) {
  const imageTypes = new Set(["image/gif", "image/jpeg", "image/png", "image/webp"]);
  const imageExtensions = new Set([".gif", ".jpeg", ".jpg", ".png", ".webp"]);

  function clipboardImageFiles(clipboardData) {
    if (!clipboardData) return [];
    const candidates = [];
    for (const item of clipboardData.items || []) {
      if (item.kind !== "file" || !(item.type || "").toLowerCase().startsWith("image/")) continue;
      const file = item.getAsFile();
      if (file) candidates.push(file);
    }
    if (!candidates.length) {
      candidates.push(...Array.from(clipboardData.files || []).filter((file) => (file.type || "").toLowerCase().startsWith("image/")));
    }
    return [...new Set(candidates)].filter((file) => {
      const extension = file.name?.slice(file.name.lastIndexOf(".")).toLowerCase();
      return imageTypes.has((file.type || "").toLowerCase()) || imageExtensions.has(extension);
    });
  }

  function changedConversations(previous, incoming, currentID) {
    const previousByID = new Map(previous.map((conversation) => [conversation.id, conversation]));
    return incoming.filter((conversation) => {
      if (!conversation.id || conversation.id === currentID) return false;
      const prior = previousByID.get(conversation.id);
      if (!prior?.updated_at || !conversation.updated_at) return false;
      const previousTime = Date.parse(prior.updated_at);
      const currentTime = Date.parse(conversation.updated_at);
      return Number.isFinite(previousTime) && Number.isFinite(currentTime) && currentTime > previousTime;
    });
  }

  function groupConversationsByRoot(conversations) {
    if (!Array.isArray(conversations) || !conversations.length) return [];
    const byID = new Map(conversations.map((item) => [item.id, item]));
    const groupsByRootID = new Map();

    const resolveRootID = (conversation) => {
      const explicitRoot = conversation.root_conversation_id || conversation.parent_conversation_id || "";
      if (explicitRoot && explicitRoot !== conversation.id) return explicitRoot;
      return conversation.id;
    };

    for (const conversation of conversations) {
      if (!conversation?.id) continue;
      const rootID = resolveRootID(conversation);
      let group = groupsByRootID.get(rootID);
      if (!group) {
        group = {
          rootId: rootID,
          root: byID.get(rootID) || null,
          children: [],
          latestTime: 0,
        };
        groupsByRootID.set(rootID, group);
      }
      const ts = Date.parse(conversation.updated_at || "") || 0;
      if (ts > group.latestTime) group.latestTime = ts;
      if (conversation.id === rootID) {
        group.root = conversation;
      } else {
        group.children.push(conversation);
      }
    }

    const groups = Array.from(groupsByRootID.values());
    for (const group of groups) {
      group.children.sort((a, b) => (Date.parse(b.updated_at || "") || 0) - (Date.parse(a.updated_at || "") || 0));
    }
    groups.sort((a, b) => b.latestTime - a.latestTime);
    return groups;
  }

  function projectConversationStats(projects, conversations, unreadIDs = new Set(), showArchived = false) {
    const stats = new Map();
    for (const project of projects || []) {
      if (project?.id) {
        stats.set(project.id, { count: 0, hasUpdate: false, hasRunning: false });
      }
    }
    for (const conv of conversations || []) {
      if (!conv?.project_id || Boolean(conv.archived) !== Boolean(showArchived)) continue;
      let entry = stats.get(conv.project_id);
      if (!entry) {
        entry = { count: 0, hasUpdate: false, hasRunning: false };
        stats.set(conv.project_id, entry);
      }
      entry.count += 1;
      if (unreadIDs?.has?.(conv.id)) entry.hasUpdate = true;
      if (conv.latest_turn_status === "running") {
        entry.hasRunning = true;
        entry.hasUpdate = true;
      }
    }
    return stats;
  }

  function filterConversations(conversations, selectedProjectIDs = new Set(), showArchived = false) {
    const filterByProject = selectedProjectIDs && selectedProjectIDs.size > 0;
    return (conversations || []).filter((item) => {
      if (Boolean(item.archived) !== Boolean(showArchived)) return false;
      if (filterByProject && !selectedProjectIDs.has(item.project_id)) return false;
      return true;
    });
  }

  const api = {
    changedConversations,
    clipboardImageFiles,
    groupConversationsByRoot,
    projectConversationStats,
    filterConversations,
  };
  global.CockpitInteractions = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(globalThis);
