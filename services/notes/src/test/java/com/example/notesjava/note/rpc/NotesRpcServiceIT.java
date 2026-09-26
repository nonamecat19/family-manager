package com.example.notesjava.note.rpc;

import com.example.notesjava.common.security.Caller;
import com.example.notesjava.common.security.CallerContext;
import com.example.notesjava.note.repository.NoteRepository;
import com.example.notesjava.support.AbstractIntegrationTest;
import com.nnc.familymanager.notes.v1.ArchiveNoteRequest;
import com.nnc.familymanager.notes.v1.Block;
import com.nnc.familymanager.notes.v1.BlockType;
import com.nnc.familymanager.notes.v1.CreateNoteRequest;
import com.nnc.familymanager.notes.v1.ListNotesRequest;
import com.nnc.familymanager.notes.v1.ToggleStarRequest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.bean.override.mockito.MockitoBean;

import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.when;

@SpringBootTest
class NotesRpcServiceIT extends AbstractIntegrationTest {

    private static final UUID FAMILY = UUID.fromString("22222222-2222-2222-2222-222222222222");
    private static final String USER = "11111111-1111-1111-1111-111111111111";

    @Autowired
    private NotesRpcService rpc;

    @Autowired
    private NoteRepository notes;

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
}
