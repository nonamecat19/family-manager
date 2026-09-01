/**
 * Every string the app renders, in one module.
 *
 * apps/recipes carries a full `I18nProvider` with per-locale translation files. Commonplace
 * does not, deliberately: this app ships one locale, and a provider whose only job is to look
 * a key up in a single table is indirection without a second table to justify it. What the
 * provider actually buys — "no literal strings scattered through the screens" — is bought here
 * too, and the day a second locale lands, this object is the table it plugs into.
 *
 * Copy tone follows the imported design: sentence case, no exclamation marks, and the app
 * never congratulates the user for typing.
 */
export const strings = {
  app: {
    name: "Commonplace",
    booting: "Opening your notes",
    restoring: "Restoring your session",
    crashed: "Commonplace lost its place",
  },

  common: {
    loadFailed: "That did not load.",
    tryAgain: "Try again",
    errorReference: (ref: string) => `Reference ${ref}`,
    cancel: "Cancel",
    close: "Close",
    save: "Save",
    remove: "Remove",
    untitled: "Untitled note",
    oneMoment: "One moment",
  },

  login: {
    signInBody: "Notes for the people you live with. Yours until you share them.",
    registerBody: "Make an account, then a family. Everything you write starts private.",
    email: "Email",
    password: "Password",
    name: "Name",
    signIn: "Sign in",
    createAccount: "Create account",
    switchToRegister: "No account yet? Create one",
    switchToLogin: "Already have an account? Sign in",
    loginError: "That sign-in did not go through.",
    registerError: "That account could not be created.",
  },

  onboarding: {
    title: "Start a family",
    body: "Notebooks, notes and shares all live inside one family. Name yours to begin.",
    familyName: "Family name",
    familyNamePlaceholder: "The Reyes house",
    create: "Create family",
    creating: "Creating",
  },

  gate: {
    settingUp: "Setting things down",
    noFamilyTitle: "No family yet",
    noFamilyBody: "Commonplace keeps notes inside a family. Make one, or accept an invitation.",
    createFamily: "Create a family",
    errorTitle: "Something went sideways",
  },

  rail: {
    search: "Search",
    newNote: "New note",
    allNotes: "All notes",
    sharedWithMe: "Shared with me",
    recent: "Recent",
    starred: "Starred",
    notebooks: "Notebooks",
    newNotebook: "New notebook",
    archive: "Archive",
    notebookActions: "Notebook actions",
    notebookName: "Notebook name",
    notebookNamePlaceholder: "Meetings",
    createNotebook: "Create notebook",
    create: "Create",
    rename: "Rename",
    renameNotebook: "Rename notebook",
    archiveNotebook: "Archive notebook",
    restoreNotebook: "Take out of the archive",
    deleteNotebook: "Delete notebook",
    deleteNotebookConfirm: "Delete this notebook? Move its notes out first — it will not take them with it.",
    notebookFailed: "That notebook change did not go through.",
    settings: "Settings",
    familyCount: (n: number) => (n === 1 ? "1 in the family" : `${n} in the family`),
  },

  list: {
    title: "All notes",
    noteCount: (n: number) => (n === 1 ? "1 note" : `${n} notes`),
    sort: "Sort",
    sortBy: "Sort by",
    sortUpdated: "Last edited",
    sortCreated: "Date created",
    sortTitle: "Title",
    filter: "Filter",
    density: "Density",
    dense: "Dense",
    cards: "Cards",
    emptyTitle: "Nothing here yet",
    emptyBody: "Notes you write start private. Share one when you want it read.",
    emptySharedTitle: "Nobody has shared a note with you",
    emptySharedBody: "When someone shares a note or a notebook, it lands here.",
    emptyArchiveTitle: "The archive is empty",
    // This list is ListNotes with archived_only, which reads the NOTE's archived flag and
    // nothing else — there is no predicate on the notebook's. So archiving a notebook does not
    // put its notes here, and the old copy ("Archived notebooks and their notes rest here")
    // promised something the server does not do. The copy now says what the filter does.
    //
    // FOLLOW-UP, server side, deliberately not worked around in the client: to make the
    // notebook half true, ListNotes would have to join notebooks and treat a note whose
    // notebook is archived as archived — roughly, alongside the existing archived_only
    // predicate in services/notes/internal/db/queries/notes.sql:
    //   LEFT JOIN notebooks nb ON nb.id = n.notebook_id
    //   ... AND (NOT archived_only OR n.archived OR nb.archived)
    // and the same term inverted in the include_archived line, so an archived notebook's notes
    // also leave "All notes". That is a change to what every list returns, so it belongs in a
    // task with its own impact check — not in a client that would have to page and filter
    // locally to fake it.
    emptyArchiveBody: "Notes you archive rest here instead of being deleted. Archiving a notebook leaves its notes where they are.",
    newNote: "New note",
    notSynced: "Not synced yet",
    pickANote: "Pick a note",
    pickANoteBody: "Choose something on the left, or start a new note.",
    today: "Today",
    thisWeek: "Earlier this week",
    older: "Older",
  },

  note: {
    saved: "All changes saved",
    saving: "Saving",
    unsaved: "Unsaved changes",
    saveFailed: "Could not save",
    share: "Share",
    more: "More",
    back: "Back",
    starOn: "Star this note",
    starOff: "Unstar this note",
    delete: "Delete note",
    deleteConfirm: "Delete this note? It does not come back.",
    deleteFailed: "That note could not be deleted. Only its owner can.",
    moveTo: "Move to notebook",
    noNotebook: "No notebook",
    inThisNote: "In this note",
    activity: "Activity",
    sharedWith: "Shared with",
    comments: "Comments",
    addComment: "Add a comment",
    commentPlaceholder: "Leave a comment",
    resolve: "Resolve",
    resolved: "Resolved",
    owner: "Owner",
    canEdit: "Can edit",
    canView: "Can view",
    readOnly: "You can read this note, not edit it.",
    placeholder: "Type. “# ” for a heading, “- ” for a list, “[] ” for a task.",
    // The label on an image block written by a client that still had them. This app cannot
    // make one and does not show the picture — see BlockEditor — but the block is still here.
    imageBlock: "Image — not shown in this version",
    titlePlaceholder: "Untitled",
    created: (when: string) => `Created ${when}`,
    version: (v: number) => `v${v}`,
    editedBy: (who: string, when: string) => `${who} edited ${when}`,
    updated: (when: string) => `Updated ${when}`,
    conflictTitle: "This note changed elsewhere",
    conflictBody:
      "Someone saved a newer version while you were writing. Reload theirs, or overwrite it with yours.",
    conflictReload: "Reload theirs",
    conflictOverwrite: "Overwrite with mine",
    notFound: "That note is gone",
    notFoundBody: "It was deleted, or it was never shared with you.",
    done: "Done",
  },

  blockBar: {
    text: "Text",
    heading: "Heading",
    todo: "Task",
    bullet: "Bulleted list",
    numbered: "Numbered list",
    quote: "Quote",
    code: "Code",
    divider: "Divider",
    comment: "Comment",
  },

  gutter: {
    insert: "Insert a block below",
    moveUp: "Move this block up",
    moveDown: "Move this block down",
  },

  activity: {
    created: (who: string) => `${who} created this note`,
    edited: (who: string) => `${who} edited this note`,
    taskChecked: (who: string, what: string) =>
      what ? `${who} checked off “${what}”` : `${who} checked off a task`,
    taskUnchecked: (who: string, what: string) =>
      what ? `${who} unchecked “${what}”` : `${who} unchecked a task`,
    commented: (who: string) => `${who} commented`,
    shared: (who: string, what: string) =>
      what ? `${who} shared this with ${what}` : `${who} shared this note`,
    unknown: (who: string) => `${who} did something here`,
    empty: "Nothing has happened here yet.",
  },

  palette: {
    placeholder: "Search notes, tasks and notebooks",
    esc: "esc",
    facetAll: "All",
    facetNotes: "Notes",
    facetTasks: "Tasks",
    facetNotebooks: "Notebooks",
    groupNotes: "Notes",
    groupTasks: "Tasks",
    groupNotebooks: "Notebooks & actions",
    createNote: (q: string) => `Create note “${q}”`,
    navigate: "navigate",
    filter: "filter",
    newNote: "new note",
    footer: (notes: number, ms: number) => `Searched ${notes} notes in ${ms}ms`,
    empty: "Nothing matched.",
    hint: "Type to search across every notebook.",
    open: "Open search",
  },

  share: {
    title: "Share this note",
    titleNotebook: "Share this notebook",
    body: "Notes are private until you share them. Sharing a notebook shares everything in it.",
    wholeFamily: "Everyone in the family",
    view: "View",
    edit: "Edit",
    notShared: "Not shared",
    shares: "Shared with",
    removeShare: "Stop sharing",
    noMembers: "There is nobody else in this family yet.",
    inviteHint: "Invite people to the family from Settings.",
  },

  capture: {
    title: "Quick capture",
    offline: "Offline — saves locally",
    online: "Saves to the family",
    queued: (n: number) => (n === 1 ? "1 note waiting to sync" : `${n} notes waiting to sync`),
    titlePlaceholder: "Title",
    bodyPlaceholder: "Anything worth keeping",
    kindNote: "Note",
    kindTask: "Task",
    noNotebook: "No notebook",
    save: "Save",
    saved: "Saved",
    open: "Quick capture",
  },

  tabs: {
    notes: "Notes",
    search: "Search",
    shared: "Shared",
    settings: "Settings",
  },

  search: {
    title: "Search",
    placeholder: "Search notes, tasks and notebooks",
    empty: "Nothing matched.",
    hint: "Search runs across every notebook you can see.",
  },

  settings: {
    title: "Settings",
    family: "Family",
    members: "Members",
    account: "Account",
    signOut: "Sign out",
    offlineQueue: "Waiting to sync",
    flushNow: "Sync now",
    queueEmpty: "Everything is synced.",
    about: "About",
    aboutBody: "Commonplace — notes for one family. Blocks, notebooks, and a search box.",
  },
} as const;
