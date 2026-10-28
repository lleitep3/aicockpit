const test = require("node:test");
const assert = require("node:assert/strict");

const {
  changedConversations,
  clipboardImageFiles,
  filterConversations,
  groupConversationsByRoot,
  projectConversationStats,
} = require("./interaction.js");

test("clipboard image extraction accepts supported image files and avoids duplicates", () => {
  const png = { name: "", type: "image/png" };
  const clipboardData = {
    items: [{ kind: "file", type: "image/png", getAsFile: () => png }],
    files: [png],
  };

  assert.deepEqual(clipboardImageFiles(clipboardData), [png]);
});

test("clipboard image extraction rejects unsupported and non-image clipboard entries", () => {
  const unsupported = { name: "capture.bmp", type: "image/bmp" };
  const text = { name: "notes.txt", type: "text/plain" };
  const clipboardData = {
    items: [
      { kind: "file", type: unsupported.type, getAsFile: () => unsupported },
      { kind: "string", type: text.type, getAsFile: () => text },
    ],
  };

  assert.deepEqual(clipboardImageFiles(clipboardData), []);
});

test("conversation update detection ignores the selected chat and unchanged timestamps", () => {
  const previous = [
    { id: "selected", updated_at: "2026-10-27T12:00:00Z" },
    { id: "unchanged", updated_at: "2026-10-27T12:00:00Z" },
    { id: "updated", updated_at: "2026-10-27T12:00:00Z" },
  ];
  const incoming = [
    { id: "selected", updated_at: "2026-10-27T12:01:00Z" },
    { id: "unchanged", updated_at: "2026-10-27T12:00:00Z" },
    { id: "updated", updated_at: "2026-10-27T12:02:00Z" },
  ];

  assert.deepEqual(changedConversations(previous, incoming, "selected"), [incoming[2]]);
});

test("groupConversationsByRoot groups spawned child sessions under their root conversation", () => {
  const conversations = [
    { id: "conv_unrelated", title: "Outra conversa", updated_at: "2026-10-27T21:25:00Z" },
    {
      id: "conv_child_copilot",
      title: "Análise (Copilot)",
      parent_conversation_id: "conv_root",
      root_conversation_id: "conv_root",
      updated_at: "2026-10-27T21:28:00Z",
    },
    {
      id: "conv_child_devin",
      title: "Análise (Devin)",
      parent_conversation_id: "conv_root",
      root_conversation_id: "conv_root",
      updated_at: "2026-10-27T21:27:00Z",
    },
    { id: "conv_root", title: "Conversa Raiz", updated_at: "2026-10-27T21:26:00Z" },
  ];

  const groups = groupConversationsByRoot(conversations);
  assert.equal(groups.length, 2);
  assert.equal(groups[0].rootId, "conv_root");
  assert.equal(groups[0].root.id, "conv_root");
  assert.deepEqual(
    groups[0].children.map((c) => c.id),
    ["conv_child_copilot", "conv_child_devin"],
  );
  assert.equal(groups[1].rootId, "conv_unrelated");
  assert.equal(groups[1].children.length, 0);
});

test("projectConversationStats calculates conversation counts and update signals per project", () => {
  const projects = [{ id: "proj_a" }, { id: "proj_b" }];
  const conversations = [
    { id: "c1", project_id: "proj_a", archived: false, latest_turn_status: "completed" },
    { id: "c2", project_id: "proj_a", archived: false, latest_turn_status: "running" },
    { id: "c3", project_id: "proj_a", archived: true, latest_turn_status: "completed" },
    { id: "c4", project_id: "proj_b", archived: false, latest_turn_status: "completed" },
  ];
  const unread = new Set(["c4"]);

  const activeStats = projectConversationStats(projects, conversations, unread, false);
  assert.deepEqual(activeStats.get("proj_a"), {
    count: 2,
    hasUpdate: true,
    hasRunning: true,
  });
  assert.deepEqual(activeStats.get("proj_b"), {
    count: 1,
    hasUpdate: true,
    hasRunning: false,
  });

  const archivedStats = projectConversationStats(projects, conversations, unread, true);
  assert.deepEqual(archivedStats.get("proj_a"), {
    count: 1,
    hasUpdate: false,
    hasRunning: false,
  });
  assert.deepEqual(archivedStats.get("proj_b"), {
    count: 0,
    hasUpdate: false,
    hasRunning: false,
  });
});

test("filterConversations filters by archived flag and multiple selected project checkboxes", () => {
  const conversations = [
    { id: "c1", project_id: "proj_a", archived: false },
    { id: "c2", project_id: "proj_b", archived: false },
    { id: "c3", project_id: "proj_a", archived: true },
    { id: "c4", project_id: "", archived: false },
  ];

  assert.deepEqual(
    filterConversations(conversations, new Set(), false).map((c) => c.id),
    ["c1", "c2", "c4"],
  );
  assert.deepEqual(
    filterConversations(conversations, new Set(["proj_a", "proj_b"]), false).map((c) => c.id),
    ["c1", "c2"],
  );
  assert.deepEqual(
    filterConversations(conversations, new Set(["proj_a"]), true).map((c) => c.id),
    ["c3"],
  );
});
