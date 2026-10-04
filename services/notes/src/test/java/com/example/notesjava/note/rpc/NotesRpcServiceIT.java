package com.example.notesjava.note.rpc;

import com.example.notesjava.common.security.Caller;
import com.example.notesjava.common.security.CallerContext;
import com.example.notesjava.note.repository.NoteRepository;
import com.example.notesjava.support.AbstractIntegrationTest;
import com.example.notesjava.common.connect.ConnectCode;
import com.example.notesjava.common.connect.ConnectException;
import com.nnc.familymanager.notes.v1.Activity;
import com.nnc.familymanager.notes.v1.ActivityKind;
import com.nnc.familymanager.notes.v1.AddCommentRequest;
import com.nnc.familymanager.notes.v1.AddCommentResponse;
import com.nnc.familymanager.notes.v1.ArchiveNoteRequest;
import com.nnc.familymanager.notes.v1.Block;
import com.nnc.familymanager.notes.v1.BlockType;
import com.nnc.familymanager.notes.v1.Comment;
import com.nnc.familymanager.notes.v1.CreateNoteRequest;
import com.nnc.familymanager.notes.v1.CreateNotebookRequest;
import com.nnc.familymanager.notes.v1.CreateNotebookResponse;
import com.nnc.familymanager.notes.v1.DeleteNotebookRequest;
import com.nnc.familymanager.notes.v1.GetNoteRequest;
import com.nnc.familymanager.notes.v1.GetNoteResponse;
import com.nnc.familymanager.notes.v1.ListSharedWithMeRequest;
import com.nnc.familymanager.notes.v1.ListSharedWithMeResponse;
import com.nnc.familymanager.notes.v1.ListSharesRequest;
import com.nnc.familymanager.notes.v1.ListSharesResponse;
import com.nnc.familymanager.notes.v1.MoveNoteRequest;
import com.nnc.familymanager.notes.v1.SearchRequest;
import com.nnc.familymanager.notes.v1.SearchResponse;
import com.nnc.familymanager.notes.v1.Share;
import com.nnc.familymanager.notes.v1.ShareNoteResponse;
import com.nnc.familymanager.notes.v1.ShareNotebookRequest;
import com.nnc.familymanager.notes.v1.ShareNotebookResponse;
import com.nnc.familymanager.notes.v1.UnshareRequest;
import com.nnc.familymanager.notes.v1.UploadNoteImageRequest;
import com.nnc.familymanager.notes.v1.DeleteNoteRequest;
import com.nnc.familymanager.notes.v1.ListActivityRequest;
import com.nnc.familymanager.notes.v1.ListActivityResponse;
import com.nnc.familymanager.notes.v1.ListCommentsRequest;
import com.nnc.familymanager.notes.v1.ListCommentsResponse;
import com.nnc.familymanager.notes.v1.ListNotesRequest;
import com.nnc.familymanager.notes.v1.ResolveCommentRequest;
import com.nnc.familymanager.notes.v1.ResolveCommentResponse;
import com.nnc.familymanager.notes.v1.ShareNoteRequest;
import com.nnc.familymanager.notes.v1.SharePermission;
import com.nnc.familymanager.notes.v1.ShareSubject;
import com.nnc.familymanager.notes.v1.ToggleStarRequest;
import com.nnc.familymanager.notes.v1.UpdateNoteRequest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.bean.override.mockito.MockitoBean;

import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.catchThrowableOfType;
import static org.mockito.Mockito.when;

@SpringBootTest
class NotesRpcServiceIT extends AbstractIntegrationTest {

    private static final UUID FAMILY = UUID.fromString("22222222-2222-2222-2222-222222222222");
    private static final String USER = "11111111-1111-1111-1111-111111111111";
    private static final String OTHER = "33333333-3333-3333-3333-333333333333";

    @Autowired
    private NotesRpcService rpc;

    @Autowired
    private NoteRepository notes;

    @Autowired
    private com.example.notesjava.note.repository.NoteCommentRepository commentRepository;

    @Autowired
    private com.example.notesjava.note.repository.NoteActivityRepository activityRepository;

    @MockitoBean
    private CallerContext callers;

    @BeforeEach
    void caller() {
        when(callers.require()).thenReturn(new Caller(USER, FAMILY, "ada@example.test"));
        when(callers.requireFamilyId()).thenReturn(FAMILY);
    }

    private String createNote(String title) {
        return ((com.nnc.familymanager.notes.v1.CreateNoteResponse) rpc.procedure("CreateNote").invoke(
                CreateNoteRequest.newBuilder()
                        .setTitle(title)
                        .addBlocks(Block.newBuilder()
                                .setId("b1").setType(BlockType.BLOCK_TYPE_PARAGRAPH).setText("body of " + title))
                        .build())).getNote().getId();
    }

    @Test
    void starringANoteIsWrittenToTheDatabase() {
        String noteId = createNote("starred");

        rpc.procedure("ToggleStar").invoke(ToggleStarRequest.newBuilder().setNoteId(noteId).build());

        assertThat(notes.findById(Long.valueOf(noteId)).orElseThrow().isStarred())
                .as("dispatch runs outside Spring's transactional proxy, so the write must be committed here")
                .isTrue();
    }

    @Test
    void archivingANoteIsWrittenToTheDatabaseAndHidesIt() {
        String noteId = createNote("archived");

        rpc.procedure("ArchiveNote").invoke(
                ArchiveNoteRequest.newBuilder().setNoteId(noteId).setArchived(true).build());

        assertThat(notes.findById(Long.valueOf(noteId)).orElseThrow().isArchived()).isTrue();

        var listed = (com.nnc.familymanager.notes.v1.ListNotesResponse)
                rpc.procedure("ListNotes").invoke(ListNotesRequest.getDefaultInstance());
        assertThat(listed.getNotesList()).noneMatch(note -> note.getId().equals(noteId));

        var archivedOnly = (com.nnc.familymanager.notes.v1.ListNotesResponse) rpc.procedure("ListNotes")
                .invoke(ListNotesRequest.newBuilder().setArchivedOnly(true).build());
        assertThat(archivedOnly.getNotesList()).anyMatch(note -> note.getId().equals(noteId));
    }

    @Test
    void starredOnlyFiltersTheList() {
        String starred = createNote("keep");
        String plain = createNote("ignore");

        rpc.procedure("ToggleStar").invoke(ToggleStarRequest.newBuilder().setNoteId(starred).build());

        var listed = (com.nnc.familymanager.notes.v1.ListNotesResponse) rpc.procedure("ListNotes")
                .invoke(ListNotesRequest.newBuilder().setStarredOnly(true).build());

        assertThat(listed.getNotesList()).extracting(com.nnc.familymanager.notes.v1.Note::getId)
                .contains(starred)
                .doesNotContain(plain);
    }

    private void actAs(String userId) {
        when(callers.require()).thenReturn(new Caller(userId, FAMILY, userId + "@example.test"));
    }

    private Comment comment(String noteId, String body) {
        return ((AddCommentResponse) rpc.procedure("AddComment").invoke(
                AddCommentRequest.newBuilder().setNoteId(noteId).setBody(body).build())).getComment();
    }

    private java.util.List<Comment> comments(String noteId, boolean includeResolved) {
        return ((ListCommentsResponse) rpc.procedure("ListComments").invoke(
                ListCommentsRequest.newBuilder().setNoteId(noteId).setIncludeResolved(includeResolved).build()))
                .getCommentsList();
    }

    private java.util.List<Activity> activity(String noteId) {
        return ((ListActivityResponse) rpc.procedure("ListActivity").invoke(
                ListActivityRequest.newBuilder().setNoteId(noteId).build())).getActivityList();
    }

    @Test
    void aCommentIsStoredAndListedOldestFirst() {
        String noteId = createNote("discussed");

        Comment first = comment(noteId, "  first  ");
        comment(noteId, "second");

        assertThat(first.getBody()).isEqualTo("first");
        assertThat(first.getAuthorUserId()).isEqualTo(USER);
        assertThat(first.getNoteId()).isEqualTo(noteId);
        assertThat(first.hasCreatedAt()).isTrue();
        assertThat(comments(noteId, false)).extracting(Comment::getBody).containsExactly("first", "second");
    }

    @Test
    void aBlankCommentIsRefused() {
        String noteId = createNote("quiet");

        ConnectException refused = catchThrowableOfType(ConnectException.class, () -> comment(noteId, "   "));

        assertThat(refused.code()).isEqualTo(ConnectCode.INVALID_ARGUMENT);
    }

    @Test
    void aCommentOnAnotherFamilysNoteIsNotFound() {
        String noteId = createNote("private");
        when(callers.require()).thenReturn(
                new Caller(USER, UUID.fromString("44444444-4444-4444-4444-444444444444"), "ada@example.test"));

        ConnectException refused = catchThrowableOfType(ConnectException.class, () -> comment(noteId, "hi"));

        assertThat(refused.code()).isEqualTo(ConnectCode.NOT_FOUND);
    }

    @Test
    void aResolvedCommentLeavesTheDefaultList() {
        String noteId = createNote("resolved");
        Comment open = comment(noteId, "fix the typo");

        var resolved = (ResolveCommentResponse) rpc.procedure("ResolveComment").invoke(
                ResolveCommentRequest.newBuilder().setCommentId(open.getId()).setResolved(true).build());

        assertThat(resolved.getComment().getResolved()).isTrue();
        assertThat(comments(noteId, false)).isEmpty();
        assertThat(comments(noteId, true)).extracting(Comment::getId).containsExactly(open.getId());
    }

    @Test
    void onlyTheAuthorOrTheNoteOwnerResolvesAComment() {
        String noteId = createNote("owned by USER");
        rpc.procedure("ShareNote").invoke(ShareNoteRequest.newBuilder()
                .setNoteId(noteId)
                .setSubject(ShareSubject.SHARE_SUBJECT_FAMILY)
                .setPermission(SharePermission.SHARE_PERMISSION_VIEW)
                .build());
        actAs(OTHER);
        Comment byOther = comment(noteId, "from someone else");
        actAs("55555555-5555-5555-5555-555555555555");

        ConnectException refused = catchThrowableOfType(ConnectException.class, () -> rpc.procedure("ResolveComment")
                .invoke(ResolveCommentRequest.newBuilder().setCommentId(byOther.getId()).setResolved(true).build()));
        assertThat(refused.code()).isEqualTo(ConnectCode.NOT_FOUND);

        actAs(USER);
        var resolved = (ResolveCommentResponse) rpc.procedure("ResolveComment").invoke(
                ResolveCommentRequest.newBuilder().setCommentId(byOther.getId()).setResolved(true).build());
        assertThat(resolved.getComment().getResolved()).isTrue();
    }

    @Test
    void activityRecordsCreationEditsTicksAndCommentsNewestFirst() {
        String noteId = ((com.nnc.familymanager.notes.v1.CreateNoteResponse) rpc.procedure("CreateNote").invoke(
                CreateNoteRequest.newBuilder()
                        .setTitle("Groceries")
                        .addBlocks(Block.newBuilder().setId("t1").setType(BlockType.BLOCK_TYPE_TODO).setText("Buy milk"))
                        .build())).getNote().getId();

        rpc.procedure("UpdateNote").invoke(UpdateNoteRequest.newBuilder()
                .setNoteId(noteId)
                .addBlocks(Block.newBuilder().setId("t1").setType(BlockType.BLOCK_TYPE_TODO).setText("Buy milk").setChecked(true))
                .build());
        rpc.procedure("UpdateNote").invoke(UpdateNoteRequest.newBuilder()
                .setNoteId(noteId)
                .setTitle("Groceries for Sunday")
                .addBlocks(Block.newBuilder().setId("t1").setType(BlockType.BLOCK_TYPE_TODO).setText("Buy milk").setChecked(true))
                .build());
        comment(noteId, "oat milk please");

        assertThat(activity(noteId))
                .extracting(Activity::getKind, Activity::getDetail, Activity::getActorUserId)
                .containsExactly(
                        org.assertj.core.groups.Tuple.tuple(ActivityKind.ACTIVITY_KIND_COMMENTED, "oat milk please", USER),
                        org.assertj.core.groups.Tuple.tuple(ActivityKind.ACTIVITY_KIND_EDITED, "", USER),
                        org.assertj.core.groups.Tuple.tuple(ActivityKind.ACTIVITY_KIND_TASK_CHECKED, "Buy milk", USER),
                        org.assertj.core.groups.Tuple.tuple(ActivityKind.ACTIVITY_KIND_CREATED, "", USER));
    }

    @Test
    void activityHonoursTheLimit() {
        String noteId = createNote("busy");
        comment(noteId, "one");
        comment(noteId, "two");

        var limited = (ListActivityResponse) rpc.procedure("ListActivity").invoke(
                ListActivityRequest.newBuilder().setNoteId(noteId).setLimit(1).build());

        assertThat(limited.getActivityList()).extracting(Activity::getDetail).containsExactly("two");
    }

    @Test
    void deletingANoteTakesItsCommentsAndActivityWithIt() {
        String noteId = createNote("doomed");
        comment(noteId, "bye");

        rpc.procedure("DeleteNote").invoke(DeleteNoteRequest.newBuilder().setNoteId(noteId).build());

        assertThat(commentRepository.findAll()).noneMatch(c -> c.getNoteId().equals(Long.valueOf(noteId)));
        assertThat(activityRepository.findAll()).noneMatch(a -> a.getNoteId().equals(Long.valueOf(noteId)));
    }

    @Test
    void anotherFamilySeesNoCommentsOrActivityAndCannotResolve() {
        String noteId = createNote("ours");
        Comment ours = comment(noteId, "family only");
        UUID strangers = UUID.fromString("44444444-4444-4444-4444-444444444444");
        when(callers.require()).thenReturn(new Caller(USER, strangers, "ada@example.test"));
        when(callers.requireFamilyId()).thenReturn(strangers);

        assertThat(comments(noteId, true)).isEmpty();
        assertThat(activity(noteId)).isEmpty();
        ConnectException refused = catchThrowableOfType(ConnectException.class, () -> rpc.procedure("ResolveComment")
                .invoke(ResolveCommentRequest.newBuilder().setCommentId(ours.getId()).setResolved(true).build()));
        assertThat(refused.code()).isEqualTo(ConnectCode.NOT_FOUND);
    }

    @Test
    void aDeletedNoteHasAnEmptyThreadRatherThanAnError() {
        String noteId = createNote("gone");
        comment(noteId, "soon orphaned");
        rpc.procedure("DeleteNote").invoke(DeleteNoteRequest.newBuilder().setNoteId(noteId).build());

        assertThat(comments(noteId, true)).isEmpty();
        assertThat(activity(noteId)).isEmpty();
    }
    private static final String THIRD = "55555555-5555-5555-5555-555555555555";

    private Share shareNote(String noteId, ShareSubject subject, String member, SharePermission permission) {
        return ((ShareNoteResponse) rpc.procedure("ShareNote").invoke(ShareNoteRequest.newBuilder()
                .setNoteId(noteId).setSubject(subject).setMemberUserId(member).setPermission(permission)
                .build())).getShare();
    }

    private Share shareNotebook(String notebookId, ShareSubject subject, String member,
                                SharePermission permission) {
        return ((ShareNotebookResponse) rpc.procedure("ShareNotebook").invoke(ShareNotebookRequest.newBuilder()
                .setNotebookId(notebookId).setSubject(subject).setMemberUserId(member).setPermission(permission)
                .build())).getShare();
    }

    private com.nnc.familymanager.notes.v1.Note getNote(String noteId) {
        return ((GetNoteResponse) rpc.procedure("GetNote").invoke(
                GetNoteRequest.newBuilder().setNoteId(noteId).build())).getNote();
    }

    private java.util.List<String> listedIds(ListNotesRequest request) {
        return ((com.nnc.familymanager.notes.v1.ListNotesResponse) rpc.procedure("ListNotes").invoke(request))
                .getNotesList().stream().map(com.nnc.familymanager.notes.v1.Note::getId).toList();
    }

    private java.util.List<Share> sharesOf(ListSharesRequest request) {
        return ((ListSharesResponse) rpc.procedure("ListShares").invoke(request)).getSharesList();
    }

    private ConnectCode refusal(Runnable call) {
        return catchThrowableOfType(ConnectException.class, call::run).code();
    }

    private String createNotebook(String name) {
        return ((CreateNotebookResponse) rpc.procedure("CreateNotebook").invoke(
                CreateNotebookRequest.newBuilder().setName(name).build())).getNotebook().getId();
    }

    private String createNoteIn(String notebookId, String title) {
        return ((com.nnc.familymanager.notes.v1.CreateNoteResponse) rpc.procedure("CreateNote").invoke(
                CreateNoteRequest.newBuilder().setNotebookId(notebookId).setTitle(title).build())).getNote().getId();
    }

    @Test
    void aNewNoteIsPrivateToItsOwner() {
        String noteId = createNote("diary entry");
        assertThat(getNote(noteId).getCanEdit()).isTrue();
        assertThat(getNote(noteId).getShared()).isFalse();

        actAs(OTHER);

        assertThat(refusal(() -> getNote(noteId))).isEqualTo(ConnectCode.NOT_FOUND);
        assertThat(listedIds(ListNotesRequest.getDefaultInstance())).doesNotContain(noteId);
        var search = (SearchResponse) rpc.procedure("Search").invoke(
                SearchRequest.newBuilder().setQuery("diary entry").build());
        assertThat(search.getHitsList()).noneMatch(hit -> hit.getNoteId().equals(noteId));
        assertThat(comments(noteId, true)).isEmpty();
        assertThat(activity(noteId)).isEmpty();
    }

    @Test
    void aViewShareLetsAMemberReadButNotWrite() {
        String noteId = createNote("shopping plan");
        Share share = shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER,
                SharePermission.SHARE_PERMISSION_VIEW);

        assertThat(share.getSubject()).isEqualTo(ShareSubject.SHARE_SUBJECT_MEMBER);
        assertThat(share.getMemberUserId()).isEqualTo(OTHER);
        assertThat(share.getPermission()).isEqualTo(SharePermission.SHARE_PERMISSION_VIEW);
        assertThat(share.getGrantedByUserId()).isEqualTo(USER);
        assertThat(share.hasCreatedAt()).isTrue();
        assertThat(activity(noteId)).extracting(Activity::getKind, Activity::getDetail)
                .contains(org.assertj.core.groups.Tuple.tuple(ActivityKind.ACTIVITY_KIND_SHARED, OTHER));

        actAs(OTHER);
        var seen = getNote(noteId);
        assertThat(seen.getCanEdit()).isFalse();
        assertThat(seen.getShared()).isTrue();
        assertThat(seen.getSharesList()).extracting(Share::getId).containsExactly(share.getId());
        assertThat(listedIds(ListNotesRequest.newBuilder().setSharedOnly(true).build())).contains(noteId);
        assertThat(((ListSharedWithMeResponse) rpc.procedure("ListSharedWithMe")
                .invoke(ListSharedWithMeRequest.getDefaultInstance())).getNotesList())
                .extracting(com.nnc.familymanager.notes.v1.Note::getId).contains(noteId);

        assertThat(refusal(() -> rpc.procedure("UpdateNote").invoke(
                UpdateNoteRequest.newBuilder().setNoteId(noteId).setTitle("mine now").build())))
                .isEqualTo(ConnectCode.PERMISSION_DENIED);
        assertThat(refusal(() -> rpc.procedure("ToggleStar").invoke(
                ToggleStarRequest.newBuilder().setNoteId(noteId).build())))
                .isEqualTo(ConnectCode.PERMISSION_DENIED);
        assertThat(comment(noteId, "looks good").getAuthorUserId()).isEqualTo(OTHER);

        actAs(THIRD);
        assertThat(refusal(() -> getNote(noteId))).isEqualTo(ConnectCode.NOT_FOUND);
    }

    @Test
    void resharingChangesThePermissionInsteadOfStackingAGrant() {
        String noteId = createNote("recipes to try");
        Share first = shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER,
                SharePermission.SHARE_PERMISSION_VIEW);
        Share second = shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER,
                SharePermission.SHARE_PERMISSION_EDIT);

        assertThat(second.getId()).isEqualTo(first.getId());
        assertThat(sharesOf(ListSharesRequest.newBuilder().setNoteId(noteId).build()))
                .extracting(Share::getPermission)
                .containsExactly(SharePermission.SHARE_PERMISSION_EDIT);

        actAs(OTHER);
        var updated = (com.nnc.familymanager.notes.v1.UpdateNoteResponse) rpc.procedure("UpdateNote").invoke(
                UpdateNoteRequest.newBuilder().setNoteId(noteId).setTitle("recipes we tried").build());
        assertThat(updated.getNote().getTitle()).isEqualTo("recipes we tried");
        assertThat(updated.getNote().getCanEdit()).isTrue();
        assertThat(sharesOf(ListSharesRequest.newBuilder().setNoteId(noteId).build())).hasSize(1);
    }

    @Test
    void concurrentSharesOfTheSameTargetConvergeOnOneGrant() throws Exception {
        String noteId = createNote("raced note");
        String notebookId = createNotebook("Raced notebook");
        int callers = 8;
        java.util.concurrent.ExecutorService pool = java.util.concurrent.Executors.newFixedThreadPool(callers);
        try {
            java.util.concurrent.CyclicBarrier start = new java.util.concurrent.CyclicBarrier(callers);
            java.util.List<java.util.concurrent.Future<Share>> noteShares = new java.util.ArrayList<>();
            java.util.List<java.util.concurrent.Future<Share>> notebookShares = new java.util.ArrayList<>();
            for (int i = 0; i < callers; i++) {
                SharePermission permission = i % 2 == 0
                        ? SharePermission.SHARE_PERMISSION_VIEW
                        : SharePermission.SHARE_PERMISSION_EDIT;
                noteShares.add(pool.submit(() -> {
                    start.await();
                    return shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER, permission);
                }));
            }
            for (int i = 0; i < callers; i++) {
                notebookShares.add(pool.submit(() -> {
                    start.await();
                    return shareNotebook(notebookId, ShareSubject.SHARE_SUBJECT_FAMILY, "",
                            SharePermission.SHARE_PERMISSION_VIEW);
                }));
            }

            java.util.Set<String> noteShareIds = new java.util.HashSet<>();
            for (var share : noteShares) {
                noteShareIds.add(share.get(30, java.util.concurrent.TimeUnit.SECONDS).getId());
            }
            java.util.Set<String> notebookShareIds = new java.util.HashSet<>();
            for (var share : notebookShares) {
                notebookShareIds.add(share.get(30, java.util.concurrent.TimeUnit.SECONDS).getId());
            }

            assertThat(noteShareIds).hasSize(1);
            assertThat(notebookShareIds).hasSize(1);
            assertThat(sharesOf(ListSharesRequest.newBuilder().setNoteId(noteId).build()))
                    .extracting(Share::getId).containsExactlyElementsOf(noteShareIds);
            assertThat(sharesOf(ListSharesRequest.newBuilder().setNotebookId(notebookId).build()))
                    .extracting(Share::getId).containsExactlyElementsOf(notebookShareIds);
        } finally {
            pool.shutdownNow();
        }
    }

    @Test
    void resharingReturnsTheNewPermissionInTheSameCall() {
        String noteId = createNote("reshare echo");
        shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER, SharePermission.SHARE_PERMISSION_VIEW);

        Share again = shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER,
                SharePermission.SHARE_PERMISSION_EDIT);

        assertThat(again.getPermission()).isEqualTo(SharePermission.SHARE_PERMISSION_EDIT);
    }

    @Test
    void searchNamesTheNotebookOnlyWhenTheViewerCanSeeIt() {
        String notebookId = createNotebook("Secret santa plans");
        String noteId = createNoteIn(notebookId, "quokka gift ideas");
        shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER, SharePermission.SHARE_PERMISSION_VIEW);

        var ownersSearch = (SearchResponse) rpc.procedure("Search").invoke(
                SearchRequest.newBuilder().setQuery("quokka").build());
        assertThat(ownersSearch.getHitsList()).filteredOn(hit -> hit.getNoteId().equals(noteId))
                .singleElement()
                .satisfies(hit -> assertThat(hit.getContext()).isEqualTo("Secret santa plans"));

        actAs(OTHER);
        var membersSearch = (SearchResponse) rpc.procedure("Search").invoke(
                SearchRequest.newBuilder().setQuery("quokka").build());
        assertThat(membersSearch.getHitsList()).filteredOn(hit -> hit.getNoteId().equals(noteId))
                .singleElement()
                .satisfies(hit -> assertThat(hit.getContext()).isEmpty());

        actAs(USER);
        shareNotebook(notebookId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER, SharePermission.SHARE_PERMISSION_VIEW);
        actAs(OTHER);
        var afterNotebookShare = (SearchResponse) rpc.procedure("Search").invoke(
                SearchRequest.newBuilder().setQuery("quokka").build());
        assertThat(afterNotebookShare.getHitsList()).filteredOn(hit -> hit.getNoteId().equals(noteId))
                .singleElement()
                .satisfies(hit -> assertThat(hit.getContext()).isEqualTo("Secret santa plans"));
    }

    @Test
    void aFamilyShareReachesEveryMemberButNoOtherFamily() {
        String noteId = createNote("holiday plans");
        shareNote(noteId, ShareSubject.SHARE_SUBJECT_FAMILY, "", SharePermission.SHARE_PERMISSION_VIEW);

        actAs(THIRD);
        assertThat(getNote(noteId).getTitle()).isEqualTo("holiday plans");

        UUID strangers = UUID.fromString("44444444-4444-4444-4444-444444444444");
        when(callers.require()).thenReturn(new Caller(THIRD, strangers, "x@example.test"));
        assertThat(refusal(() -> getNote(noteId))).isEqualTo(ConnectCode.NOT_FOUND);
        assertThat(refusal(() -> sharesOf(ListSharesRequest.newBuilder().setNoteId(noteId).build())))
                .isEqualTo(ConnectCode.NOT_FOUND);
    }

    @Test
    void aNotebookShareReachesOnlyTheNotesItsOwnerKeepsThere() {
        String notebookId = createNotebook("Household");
        String ownersNote = createNoteIn(notebookId, "bills");
        shareNotebook(notebookId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER, SharePermission.SHARE_PERMISSION_EDIT);

        assertThat(activity(ownersNote)).extracting(Activity::getKind).contains(ActivityKind.ACTIVITY_KIND_SHARED);

        actAs(OTHER);
        assertThat(listedIds(ListNotesRequest.newBuilder().setNotebookId(notebookId).build()))
                .containsExactly(ownersNote);
        assertThat(getNote(ownersNote).getCanEdit()).isTrue();
        assertThat(getNote(ownersNote).getShared()).isTrue();
        var sharedWithMe = (ListSharedWithMeResponse) rpc.procedure("ListSharedWithMe")
                .invoke(ListSharedWithMeRequest.getDefaultInstance());
        assertThat(sharedWithMe.getNotebooksList()).extracting(com.nnc.familymanager.notes.v1.Notebook::getId)
                .contains(notebookId);
        String othersNote = createNoteIn(notebookId, "my private bit");

        actAs(USER);
        shareNotebook(notebookId, ShareSubject.SHARE_SUBJECT_FAMILY, "", SharePermission.SHARE_PERMISSION_VIEW);
        assertThat(refusal(() -> getNote(othersNote))).isEqualTo(ConnectCode.NOT_FOUND);

        actAs(THIRD);
        assertThat(listedIds(ListNotesRequest.newBuilder().setNotebookId(notebookId).build()))
                .containsExactly(ownersNote);
        assertThat(getNote(ownersNote).getCanEdit()).isFalse();
        assertThat(refusal(() -> createNoteIn(notebookId, "not allowed")))
                .isEqualTo(ConnectCode.PERMISSION_DENIED);
        var notebooks = (com.nnc.familymanager.notes.v1.ListNotebooksResponse) rpc.procedure("ListNotebooks")
                .invoke(com.nnc.familymanager.notes.v1.ListNotebooksRequest.getDefaultInstance());
        assertThat(notebooks.getNotebooksList())
                .filteredOn(notebook -> notebook.getId().equals(notebookId))
                .singleElement()
                .satisfies(notebook -> {
                    assertThat(notebook.getNoteCount()).isEqualTo(1);
                    assertThat(notebook.getOwnerUserId()).isEqualTo(USER);
                });
    }

    @Test
    void onlyTheOwnerSharesMovesOrDeletes() {
        String noteId = createNote("owned");
        String notebookId = createNotebook("Owned book");
        shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER, SharePermission.SHARE_PERMISSION_EDIT);
        shareNotebook(notebookId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER, SharePermission.SHARE_PERMISSION_EDIT);

        actAs(OTHER);
        assertThat(refusal(() -> shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, THIRD,
                SharePermission.SHARE_PERMISSION_VIEW))).isEqualTo(ConnectCode.PERMISSION_DENIED);
        assertThat(refusal(() -> shareNotebook(notebookId, ShareSubject.SHARE_SUBJECT_FAMILY, "",
                SharePermission.SHARE_PERMISSION_VIEW))).isEqualTo(ConnectCode.PERMISSION_DENIED);
        assertThat(refusal(() -> rpc.procedure("DeleteNote").invoke(
                DeleteNoteRequest.newBuilder().setNoteId(noteId).build()))).isEqualTo(ConnectCode.PERMISSION_DENIED);
        assertThat(refusal(() -> rpc.procedure("MoveNote").invoke(
                MoveNoteRequest.newBuilder().setNoteId(noteId).setNotebookId(notebookId).build())))
                .isEqualTo(ConnectCode.PERMISSION_DENIED);
        assertThat(refusal(() -> rpc.procedure("DeleteNotebook").invoke(
                DeleteNotebookRequest.newBuilder().setNotebookId(notebookId).build())))
                .isEqualTo(ConnectCode.PERMISSION_DENIED);
    }

    @Test
    void unshareRevokesAccessAndAnswersNotFoundForAnythingElse() {
        String noteId = createNote("temporary");
        Share share = shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER,
                SharePermission.SHARE_PERMISSION_VIEW);

        actAs(OTHER);
        assertThat(refusal(() -> rpc.procedure("Unshare").invoke(UnshareRequest.newBuilder()
                .setNoteId(noteId).setShareId(share.getId()).build()))).isEqualTo(ConnectCode.NOT_FOUND);

        actAs(USER);
        assertThat(refusal(() -> rpc.procedure("Unshare").invoke(UnshareRequest.newBuilder()
                .setNoteId(noteId).setShareId("999999999").build()))).isEqualTo(ConnectCode.NOT_FOUND);
        rpc.procedure("Unshare").invoke(UnshareRequest.newBuilder()
                .setNoteId(noteId).setShareId(share.getId()).build());
        assertThat(sharesOf(ListSharesRequest.newBuilder().setNoteId(noteId).build())).isEmpty();
        assertThat(getNote(noteId).getShared()).isFalse();

        actAs(OTHER);
        assertThat(refusal(() -> getNote(noteId))).isEqualTo(ConnectCode.NOT_FOUND);
    }

    @Test
    void unshareFromANotebookRevokesItsNotes() {
        String notebookId = createNotebook("Shared then not");
        String noteId = createNoteIn(notebookId, "inside");
        Share share = shareNotebook(notebookId, ShareSubject.SHARE_SUBJECT_FAMILY, "",
                SharePermission.SHARE_PERMISSION_VIEW);
        assertThat(sharesOf(ListSharesRequest.newBuilder().setNotebookId(notebookId).build()))
                .extracting(Share::getSubject).containsExactly(ShareSubject.SHARE_SUBJECT_FAMILY);

        rpc.procedure("Unshare").invoke(UnshareRequest.newBuilder()
                .setNotebookId(notebookId).setShareId(share.getId()).build());

        actAs(OTHER);
        assertThat(refusal(() -> getNote(noteId))).isEqualTo(ConnectCode.NOT_FOUND);
    }

    @Test
    void sharingRefusesMalformedRequests() {
        String noteId = createNote("validated");

        assertThat(refusal(() -> shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, "",
                SharePermission.SHARE_PERMISSION_VIEW))).isEqualTo(ConnectCode.INVALID_ARGUMENT);
        assertThat(refusal(() -> shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, USER,
                SharePermission.SHARE_PERMISSION_VIEW))).isEqualTo(ConnectCode.INVALID_ARGUMENT);
        assertThat(refusal(() -> shareNote(noteId, ShareSubject.SHARE_SUBJECT_FAMILY, OTHER,
                SharePermission.SHARE_PERMISSION_VIEW))).isEqualTo(ConnectCode.INVALID_ARGUMENT);
        assertThat(refusal(() -> shareNote(noteId, ShareSubject.SHARE_SUBJECT_UNSPECIFIED, "",
                SharePermission.SHARE_PERMISSION_VIEW))).isEqualTo(ConnectCode.INVALID_ARGUMENT);
        assertThat(refusal(() -> sharesOf(ListSharesRequest.getDefaultInstance())))
                .isEqualTo(ConnectCode.INVALID_ARGUMENT);
        assertThat(refusal(() -> sharesOf(ListSharesRequest.newBuilder()
                .setNoteId(noteId).setNotebookId("1").build()))).isEqualTo(ConnectCode.INVALID_ARGUMENT);
    }

    @Test
    void anUnspecifiedPermissionIsReadAsView() {
        String noteId = createNote("defaults");

        Share share = shareNote(noteId, ShareSubject.SHARE_SUBJECT_MEMBER, OTHER,
                SharePermission.SHARE_PERMISSION_UNSPECIFIED);

        assertThat(share.getPermission()).isEqualTo(SharePermission.SHARE_PERMISSION_VIEW);
    }

    @Test
    void aNoteWithNoOwnerStaysVisibleToTheWholeFamily() {
        var legacy = notes.save(com.example.notesjava.note.domain.Note.of(FAMILY, "from the REST api", "body", null));
        String noteId = String.valueOf(legacy.getId());

        actAs(OTHER);

        assertThat(getNote(noteId).getCanEdit()).isTrue();
        assertThat(listedIds(ListNotesRequest.getDefaultInstance())).contains(noteId);
        assertThat(listedIds(ListNotesRequest.newBuilder().setSharedOnly(true).build())).doesNotContain(noteId);
    }

    @Test
    void uploadingAnImageWithNoStorageConfiguredIsUnavailable() {
        String noteId = createNote("pictures");

        assertThat(refusal(() -> rpc.procedure("UploadNoteImage").invoke(UploadNoteImageRequest.newBuilder()
                .setNoteId(noteId)
                .setContentType("image/png")
                .setImage(com.google.protobuf.ByteString.copyFrom(new byte[]{1, 2, 3}))
                .build()))).isEqualTo(ConnectCode.UNAVAILABLE);
    }
}
