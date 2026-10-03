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
import com.nnc.familymanager.notes.v1.DeleteNoteRequest;
import com.nnc.familymanager.notes.v1.ListActivityRequest;
import com.nnc.familymanager.notes.v1.ListActivityResponse;
import com.nnc.familymanager.notes.v1.ListCommentsRequest;
import com.nnc.familymanager.notes.v1.ListCommentsResponse;
import com.nnc.familymanager.notes.v1.ListNotesRequest;
import com.nnc.familymanager.notes.v1.ResolveCommentRequest;
import com.nnc.familymanager.notes.v1.ResolveCommentResponse;
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
}
