package com.example.notesjava.note.rpc;

import com.example.notesjava.common.connect.ConnectException;
import com.example.notesjava.common.connect.ConnectService;
import com.example.notesjava.common.security.Caller;
import com.example.notesjava.common.security.CallerContext;
import com.example.notesjava.group.domain.Group;
import com.example.notesjava.group.repository.GroupRepository;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.note.repository.NoteRepository;
import com.nnc.familymanager.notes.v1.AddCommentRequest;
import com.nnc.familymanager.notes.v1.ArchiveNoteRequest;
import com.nnc.familymanager.notes.v1.ArchiveNoteResponse;
import com.nnc.familymanager.notes.v1.Block;
import com.nnc.familymanager.notes.v1.CreateNoteRequest;
import com.nnc.familymanager.notes.v1.CreateNoteResponse;
import com.nnc.familymanager.notes.v1.CreateNotebookRequest;
import com.nnc.familymanager.notes.v1.CreateNotebookResponse;
import com.nnc.familymanager.notes.v1.DeleteNoteRequest;
import com.nnc.familymanager.notes.v1.DeleteNoteResponse;
import com.nnc.familymanager.notes.v1.DeleteNotebookRequest;
import com.nnc.familymanager.notes.v1.DeleteNotebookResponse;
import com.nnc.familymanager.notes.v1.GetNoteRequest;
import com.nnc.familymanager.notes.v1.GetNoteResponse;
import com.nnc.familymanager.notes.v1.ListActivityRequest;
import com.nnc.familymanager.notes.v1.ListCommentsRequest;
import com.nnc.familymanager.notes.v1.ListNotebooksRequest;
import com.nnc.familymanager.notes.v1.ListNotebooksResponse;
import com.nnc.familymanager.notes.v1.ListNotesRequest;
import com.nnc.familymanager.notes.v1.ListNotesResponse;
import com.nnc.familymanager.notes.v1.ListSharedWithMeRequest;
import com.nnc.familymanager.notes.v1.ListSharesRequest;
import com.nnc.familymanager.notes.v1.MoveNoteRequest;
import com.nnc.familymanager.notes.v1.MoveNoteResponse;
import com.nnc.familymanager.notes.v1.NoteSort;
import com.nnc.familymanager.notes.v1.ResolveCommentRequest;
import com.nnc.familymanager.notes.v1.SearchFacet;
import com.nnc.familymanager.notes.v1.SearchHit;
import com.nnc.familymanager.notes.v1.SearchRequest;
import com.nnc.familymanager.notes.v1.SearchResponse;
import com.nnc.familymanager.notes.v1.ShareNoteRequest;
import com.nnc.familymanager.notes.v1.ShareNotebookRequest;
import com.nnc.familymanager.notes.v1.ToggleStarRequest;
import com.nnc.familymanager.notes.v1.ToggleStarResponse;
import com.nnc.familymanager.notes.v1.UnshareRequest;
import com.nnc.familymanager.notes.v1.UpdateNoteRequest;
import com.nnc.familymanager.notes.v1.UpdateNoteResponse;
import com.nnc.familymanager.notes.v1.UpdateNotebookRequest;
import com.nnc.familymanager.notes.v1.UpdateNotebookResponse;
import com.nnc.familymanager.notes.v1.UploadNoteImageRequest;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Pageable;
import org.springframework.data.domain.Sort;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.Locale;
import java.util.UUID;

@Service
public class NotesRpcService extends ConnectService {

    private static final int DEFAULT_PAGE_SIZE = 50;
    private static final int MAX_PAGE_SIZE = 200;
    private static final int DEFAULT_SEARCH_LIMIT = 20;

    private final NoteRepository notes;
    private final GroupRepository groups;
    private final CallerContext callers;

    public NotesRpcService(NoteRepository notes, GroupRepository groups, CallerContext callers) {
        this.notes = notes;
        this.groups = groups;
        this.callers = callers;

        register("ListNotebooks", ListNotebooksRequest.getDefaultInstance(), this::listNotebooks);
        register("CreateNotebook", CreateNotebookRequest.getDefaultInstance(), this::createNotebook);
        register("UpdateNotebook", UpdateNotebookRequest.getDefaultInstance(), this::updateNotebook);
        register("DeleteNotebook", DeleteNotebookRequest.getDefaultInstance(), this::deleteNotebook);

        register("ListNotes", ListNotesRequest.getDefaultInstance(), this::listNotes);
        register("GetNote", GetNoteRequest.getDefaultInstance(), this::getNote);
        register("CreateNote", CreateNoteRequest.getDefaultInstance(), this::createNote);
        register("UpdateNote", UpdateNoteRequest.getDefaultInstance(), this::updateNote);
        register("MoveNote", MoveNoteRequest.getDefaultInstance(), this::moveNote);
        register("ToggleStar", ToggleStarRequest.getDefaultInstance(), this::toggleStar);
        register("ArchiveNote", ArchiveNoteRequest.getDefaultInstance(), this::archiveNote);
        register("DeleteNote", DeleteNoteRequest.getDefaultInstance(), this::deleteNote);
        register("Search", SearchRequest.getDefaultInstance(), this::search);

        registerUnimplemented();
    }

    @Override
    public String serviceName() {
        return "notes.v1.NotesService";
    }

    private void registerUnimplemented() {
        register("UploadNoteImage", UploadNoteImageRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("UploadNoteImage"); });
        register("ShareNote", ShareNoteRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("ShareNote"); });
        register("ShareNotebook", ShareNotebookRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("ShareNotebook"); });
        register("Unshare", UnshareRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("Unshare"); });
        register("ListShares", ListSharesRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("ListShares"); });
        register("ListSharedWithMe", ListSharedWithMeRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("ListSharedWithMe"); });
        register("AddComment", AddCommentRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("AddComment"); });
        register("ListComments", ListCommentsRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("ListComments"); });
        register("ResolveComment", ResolveCommentRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("ResolveComment"); });
        register("ListActivity", ListActivityRequest.getDefaultInstance(),
                req -> { throw ConnectException.unimplemented("ListActivity"); });
    }

    @Transactional(readOnly = true)
    ListNotebooksResponse listNotebooks(ListNotebooksRequest request) {
        UUID familyId = callers.requireFamilyId();
        ListNotebooksResponse.Builder response = ListNotebooksResponse.newBuilder();

        for (Group group : groups.findAllByFamilyId(familyId, Pageable.unpaged())) {
            response.addNotebooks(
                    NotesMapper.toProto(group, notes.countByFamilyIdAndGroupId(familyId, group.getId())));
        }
        return response.build();
    }

    @Transactional
    CreateNotebookResponse createNotebook(CreateNotebookRequest request) {
        UUID familyId = callers.requireFamilyId();
        String name = request.getName().trim();
        if (name.isEmpty()) {
            throw ConnectException.invalidArgument("name is required");
        }

        Group saved = groups.save(Group.of(familyId, name));
        return CreateNotebookResponse.newBuilder()
                .setNotebook(NotesMapper.toProto(saved, 0))
                .build();
    }

    @Transactional
    UpdateNotebookResponse updateNotebook(UpdateNotebookRequest request) {
        UUID familyId = callers.requireFamilyId();
        Group group = notebook(request.getNotebookId(), familyId);

        String name = request.getName().trim();
        if (name.isEmpty()) {
            throw ConnectException.invalidArgument("name is required");
        }
        group.rename(name, group.getColor());

        return UpdateNotebookResponse.newBuilder()
                .setNotebook(NotesMapper.toProto(group, notes.countByFamilyIdAndGroupId(familyId, group.getId())))
                .build();
    }

    @Transactional
    DeleteNotebookResponse deleteNotebook(DeleteNotebookRequest request) {
        UUID familyId = callers.requireFamilyId();
        Group group = notebook(request.getNotebookId(), familyId);

        notes.clearGroup(group.getId(), familyId);
        groups.delete(group);
        return DeleteNotebookResponse.getDefaultInstance();
    }

    @Transactional(readOnly = true)
    ListNotesResponse listNotes(ListNotesRequest request) {
        UUID familyId = callers.requireFamilyId();

        int pageSize = request.getPageSize() > 0
                ? Math.min(request.getPageSize(), MAX_PAGE_SIZE)
                : DEFAULT_PAGE_SIZE;
        Long notebookId = request.getNotebookId().isBlank() ? null : id(request.getNotebookId(), "notebook_id");

        Page<Note> page = notes.listForContract(
                familyId,
                notebookId,
                request.getStarredOnly(),
                request.getIncludeArchived() || request.getArchivedOnly(),
                request.getArchivedOnly(),
                PageRequest.of(0, pageSize, sortFor(request.getSort())));

        ListNotesResponse.Builder response = ListNotesResponse.newBuilder();
        page.forEach(note -> response.addNotes(NotesMapper.toProto(note)));
        return response.build();
    }

    @Transactional(readOnly = true)
    GetNoteResponse getNote(GetNoteRequest request) {
        UUID familyId = callers.requireFamilyId();
        return GetNoteResponse.newBuilder()
                .setNote(NotesMapper.toProto(note(request.getNoteId(), familyId)))
                .build();
    }

    @Transactional
    CreateNoteResponse createNote(CreateNoteRequest request) {
        Caller caller = callers.require();
        UUID familyId = caller.familyId();

        List<Block> blocks = request.getBlocksList();
        String title = request.getTitle().trim();
        if (title.isEmpty()) {
            title = BlockCodec.preview(blocks);
        }
        if (title.isBlank()) {
            throw ConnectException.invalidArgument("a note needs a title or at least one block");
        }

        Group group = request.getNotebookId().isBlank()
                ? null
                : notebook(request.getNotebookId(), familyId);

        Note note = Note.of(familyId, title, BlockCodec.toPlainText(blocks), group);
        note.owner(userId(caller));
        note.writeBlocks(BlockCodec.encode(blocks), BlockCodec.toPlainText(blocks));

        return CreateNoteResponse.newBuilder()
                .setNote(NotesMapper.toProto(notes.save(note)))
                .build();
    }

    @Transactional
    UpdateNoteResponse updateNote(UpdateNoteRequest request) {
        UUID familyId = callers.requireFamilyId();
        Note note = note(request.getNoteId(), familyId);

        String title = request.getTitle().trim();
        if (!title.isEmpty()) {
            note.rename(title);
        }
        List<Block> blocks = request.getBlocksList();
        note.writeBlocks(BlockCodec.encode(blocks), BlockCodec.toPlainText(blocks));

        return UpdateNoteResponse.newBuilder().setNote(NotesMapper.toProto(note)).build();
    }

    @Transactional
    MoveNoteResponse moveNote(MoveNoteRequest request) {
        UUID familyId = callers.requireFamilyId();
        Note note = note(request.getNoteId(), familyId);

        note.moveTo(request.getNotebookId().isBlank() ? null : notebook(request.getNotebookId(), familyId));
        return MoveNoteResponse.newBuilder().setNote(NotesMapper.toProto(note)).build();
    }

    @Transactional
    ToggleStarResponse toggleStar(ToggleStarRequest request) {
        UUID familyId = callers.requireFamilyId();
        Note note = note(request.getNoteId(), familyId);

        note.toggleStar();
        return ToggleStarResponse.newBuilder().setNote(NotesMapper.toProto(note)).build();
    }

    @Transactional
    ArchiveNoteResponse archiveNote(ArchiveNoteRequest request) {
        UUID familyId = callers.requireFamilyId();
        Note note = note(request.getNoteId(), familyId);

        note.archive(request.getArchived());
        return ArchiveNoteResponse.newBuilder().setNote(NotesMapper.toProto(note)).build();
    }

    @Transactional
    DeleteNoteResponse deleteNote(DeleteNoteRequest request) {
        UUID familyId = callers.requireFamilyId();
        notes.delete(note(request.getNoteId(), familyId));
        return DeleteNoteResponse.getDefaultInstance();
    }

    @Transactional(readOnly = true)
    SearchResponse search(SearchRequest request) {
        UUID familyId = callers.requireFamilyId();

        String query = request.getQuery().trim();
        if (query.isEmpty()) {
            throw ConnectException.invalidArgument("query is required");
        }

        long started = System.nanoTime();
        int limit = request.getLimit() > 0 ? Math.min(request.getLimit(), MAX_PAGE_SIZE) : DEFAULT_SEARCH_LIMIT;
        String needle = "%" + query.toLowerCase(Locale.ROOT) + "%";

        SearchResponse.Builder response = SearchResponse.newBuilder();
        List<Note> hits = notes.searchText(familyId, needle, PageRequest.of(0, limit));

        for (Note note : hits) {
            List<Block> blocks = BlockCodec.decode(note.getBlocks(), note.getContent());
            SearchHit.Builder hit = SearchHit.newBuilder()
                    .setKind(request.getFacet() == SearchFacet.SEARCH_FACET_UNSPECIFIED
                            ? SearchFacet.SEARCH_FACET_NOTES
                            : request.getFacet())
                    .setNoteId(String.valueOf(note.getId()))
                    .setTitle(note.getTitle())
                    .setSnippet(BlockCodec.preview(blocks));
            if (note.getGroup() != null) {
                hit.setNotebookId(String.valueOf(note.getGroup().getId()));
                hit.setContext(note.getGroup().getTitle());
            }
            if (note.getUpdatedAt() != null) {
                hit.setUpdatedAt(NotesMapper.timestamp(note.getUpdatedAt()));
            }
            response.addHits(hit);
        }

        return response
                .setSearchedNotes(hits.size())
                .setElapsedMs((int) ((System.nanoTime() - started) / 1_000_000))
                .build();
    }

    private Sort sortFor(NoteSort sort) {
        return switch (sort) {
            case NOTE_SORT_CREATED -> Sort.by(Sort.Direction.DESC, "createdAt");
            case NOTE_SORT_TITLE -> Sort.by(Sort.Direction.ASC, "title");
            default -> Sort.by(Sort.Direction.DESC, "updatedAt");
        };
    }

    private Note note(String noteId, UUID familyId) {
        return notes.findWithRelationsByIdAndFamilyId(id(noteId, "note_id"), familyId)
                .orElseThrow(() -> ConnectException.notFound("that note does not exist"));
    }

    private Group notebook(String notebookId, UUID familyId) {
        return groups.findByIdAndFamilyId(id(notebookId, "notebook_id"), familyId)
                .orElseThrow(() -> ConnectException.notFound("that notebook does not exist"));
    }

    private static Long id(String raw, String field) {
        try {
            return Long.valueOf(raw.trim());
        } catch (NumberFormatException ex) {
            throw ConnectException.invalidArgument(field + " is not a valid id");
        }
    }

    private static UUID userId(Caller caller) {
        try {
            return UUID.fromString(caller.userId());
        } catch (IllegalArgumentException ex) {
            return null;
        }
    }
}
