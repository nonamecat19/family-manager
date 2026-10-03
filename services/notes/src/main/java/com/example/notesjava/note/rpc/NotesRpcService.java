package com.example.notesjava.note.rpc;

import com.example.notesjava.common.connect.ConnectException;
import com.example.notesjava.common.connect.ConnectService;
import com.example.notesjava.common.security.Caller;
import com.example.notesjava.common.security.CallerContext;
import com.example.notesjava.group.domain.Group;
import com.example.notesjava.group.repository.GroupRepository;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.note.domain.NoteActivity;
import com.example.notesjava.note.domain.NoteComment;
import com.example.notesjava.note.repository.NoteActivityRepository;
import com.example.notesjava.note.repository.NoteCommentRepository;
import com.example.notesjava.note.repository.NoteRepository;
import com.nnc.familymanager.notes.v1.ActivityKind;
import com.nnc.familymanager.notes.v1.AddCommentRequest;
import com.nnc.familymanager.notes.v1.AddCommentResponse;
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
import com.nnc.familymanager.notes.v1.ListActivityResponse;
import com.nnc.familymanager.notes.v1.ListCommentsRequest;
import com.nnc.familymanager.notes.v1.ListCommentsResponse;
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
import com.nnc.familymanager.notes.v1.ResolveCommentResponse;
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
import org.springframework.data.jpa.domain.Specification;
import jakarta.persistence.criteria.Predicate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.support.TransactionTemplate;

import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.UUID;

@Service
public class NotesRpcService extends ConnectService {

    private static final int DEFAULT_PAGE_SIZE = 50;
    private static final int MAX_PAGE_SIZE = 200;
    private static final int DEFAULT_SEARCH_LIMIT = 20;
    private static final int DEFAULT_ACTIVITY_LIMIT = 50;
    private static final int MAX_ACTIVITY_LIMIT = 200;
    private static final int MAX_COMMENT_CODE_POINTS = 100_000;

    private final NoteRepository notes;
    private final GroupRepository groups;
    private final NoteCommentRepository comments;
    private final NoteActivityRepository activity;
    private final CallerContext callers;
    private final TransactionTemplate write;
    private final TransactionTemplate read;

    public NotesRpcService(NoteRepository notes, GroupRepository groups, NoteCommentRepository comments,
                           NoteActivityRepository activity, CallerContext callers,
                           PlatformTransactionManager transactions) {
        this.notes = notes;
        this.groups = groups;
        this.comments = comments;
        this.activity = activity;
        this.callers = callers;
        this.write = new TransactionTemplate(transactions);
        this.read = new TransactionTemplate(transactions);
        this.read.setReadOnly(true);

        register("ListNotebooks", ListNotebooksRequest.getDefaultInstance(), inTransaction(read, this::listNotebooks));
        register("CreateNotebook", CreateNotebookRequest.getDefaultInstance(), inTransaction(write, this::createNotebook));
        register("UpdateNotebook", UpdateNotebookRequest.getDefaultInstance(), inTransaction(write, this::updateNotebook));
        register("DeleteNotebook", DeleteNotebookRequest.getDefaultInstance(), inTransaction(write, this::deleteNotebook));

        register("ListNotes", ListNotesRequest.getDefaultInstance(), inTransaction(read, this::listNotes));
        register("GetNote", GetNoteRequest.getDefaultInstance(), inTransaction(read, this::getNote));
        register("CreateNote", CreateNoteRequest.getDefaultInstance(), inTransaction(write, this::createNote));
        register("UpdateNote", UpdateNoteRequest.getDefaultInstance(), inTransaction(write, this::updateNote));
        register("MoveNote", MoveNoteRequest.getDefaultInstance(), inTransaction(write, this::moveNote));
        register("ToggleStar", ToggleStarRequest.getDefaultInstance(), inTransaction(write, this::toggleStar));
        register("ArchiveNote", ArchiveNoteRequest.getDefaultInstance(), inTransaction(write, this::archiveNote));
        register("DeleteNote", DeleteNoteRequest.getDefaultInstance(), inTransaction(write, this::deleteNote));
        register("Search", SearchRequest.getDefaultInstance(), inTransaction(read, this::search));

        register("AddComment", AddCommentRequest.getDefaultInstance(), inTransaction(write, this::addComment));
        register("ListComments", ListCommentsRequest.getDefaultInstance(), inTransaction(read, this::listComments));
        register("ResolveComment", ResolveCommentRequest.getDefaultInstance(), inTransaction(write, this::resolveComment));
        register("ListActivity", ListActivityRequest.getDefaultInstance(), inTransaction(read, this::listActivity));

        registerUnimplemented();
    }

    @Override
    public String serviceName() {
        return "notes.v1.NotesService";
    }

    // Handlers are dispatched through method references, which never pass through Spring's
    // transactional proxy, so each one is wrapped here instead of annotated.
    private static <I extends com.google.protobuf.Message, O extends com.google.protobuf.Message>
    com.example.notesjava.common.connect.UnaryHandler<I, O> inTransaction(
            TransactionTemplate template, com.example.notesjava.common.connect.UnaryHandler<I, O> handler) {
        return request -> template.execute(status -> handler.handle(request));
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
    }

    ListNotebooksResponse listNotebooks(ListNotebooksRequest request) {
        UUID familyId = callers.requireFamilyId();
        ListNotebooksResponse.Builder response = ListNotebooksResponse.newBuilder();

        for (Group group : groups.findAllByFamilyId(familyId, Pageable.unpaged())) {
            response.addNotebooks(
                    NotesMapper.toProto(group, notes.countByFamilyIdAndGroupId(familyId, group.getId())));
        }
        return response.build();
    }

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

    DeleteNotebookResponse deleteNotebook(DeleteNotebookRequest request) {
        UUID familyId = callers.requireFamilyId();
        Group group = notebook(request.getNotebookId(), familyId);

        notes.clearGroup(group.getId(), familyId);
        groups.delete(group);
        return DeleteNotebookResponse.getDefaultInstance();
    }

    ListNotesResponse listNotes(ListNotesRequest request) {
        UUID familyId = callers.requireFamilyId();

        int pageSize = request.getPageSize() > 0
                ? Math.min(request.getPageSize(), MAX_PAGE_SIZE)
                : DEFAULT_PAGE_SIZE;
        Long notebookId = request.getNotebookId().isBlank() ? null : id(request.getNotebookId(), "notebook_id");

        Page<Note> page = notes.findAll(
                filter(familyId, notebookId, request.getStarredOnly(),
                        request.getIncludeArchived(), request.getArchivedOnly()),
                PageRequest.of(0, pageSize, sortFor(request.getSort())));

        ListNotesResponse.Builder response = ListNotesResponse.newBuilder();
        page.forEach(note -> response.addNotes(NotesMapper.toProto(note)));
        return response.build();
    }

    GetNoteResponse getNote(GetNoteRequest request) {
        UUID familyId = callers.requireFamilyId();
        return GetNoteResponse.newBuilder()
                .setNote(NotesMapper.toProto(note(request.getNoteId(), familyId)))
                .build();
    }

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
        Note saved = notes.save(note);
        record(caller, saved.getId(), ActivityKind.ACTIVITY_KIND_CREATED, "");

        return CreateNoteResponse.newBuilder()
                .setNote(NotesMapper.toProto(saved))
                .build();
    }

    UpdateNoteResponse updateNote(UpdateNoteRequest request) {
        Caller caller = callers.require();
        Note note = note(request.getNoteId(), caller.familyId());

        String title = request.getTitle().trim();
        List<Block> blocks = request.getBlocksList();
        EditKind edit = EditKind.classify(
                note.getTitle(), BlockCodec.decode(note.getBlocks(), note.getContent()),
                title.isEmpty() ? note.getTitle() : title, blocks);

        if (!title.isEmpty()) {
            note.rename(title);
        }
        note.writeBlocks(BlockCodec.encode(blocks), BlockCodec.toPlainText(blocks));
        record(caller, note.getId(), edit.kind(), edit.detail());

        return UpdateNoteResponse.newBuilder().setNote(NotesMapper.toProto(note)).build();
    }

    MoveNoteResponse moveNote(MoveNoteRequest request) {
        UUID familyId = callers.requireFamilyId();
        Note note = note(request.getNoteId(), familyId);

        note.moveTo(request.getNotebookId().isBlank() ? null : notebook(request.getNotebookId(), familyId));
        return MoveNoteResponse.newBuilder().setNote(NotesMapper.toProto(note)).build();
    }

    ToggleStarResponse toggleStar(ToggleStarRequest request) {
        UUID familyId = callers.requireFamilyId();
        Note note = note(request.getNoteId(), familyId);

        note.toggleStar();
        return ToggleStarResponse.newBuilder().setNote(NotesMapper.toProto(note)).build();
    }

    ArchiveNoteResponse archiveNote(ArchiveNoteRequest request) {
        UUID familyId = callers.requireFamilyId();
        Note note = note(request.getNoteId(), familyId);

        note.archive(request.getArchived());
        return ArchiveNoteResponse.newBuilder().setNote(NotesMapper.toProto(note)).build();
    }

    DeleteNoteResponse deleteNote(DeleteNoteRequest request) {
        UUID familyId = callers.requireFamilyId();
        notes.delete(note(request.getNoteId(), familyId));
        return DeleteNoteResponse.getDefaultInstance();
    }

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

    AddCommentResponse addComment(AddCommentRequest request) {
        Caller caller = callers.require();
        Note note = note(request.getNoteId(), caller.familyId());

        String body = request.getBody().trim();
        if (body.isEmpty()) {
            throw ConnectException.invalidArgument("body is required");
        }
        if (body.codePointCount(0, body.length()) > MAX_COMMENT_CODE_POINTS) {
            throw ConnectException.invalidArgument(
                    "a comment holds at most " + MAX_COMMENT_CODE_POINTS + " characters");
        }

        NoteComment comment = comments.save(
                NoteComment.of(caller.familyId(), note.getId(), requireUserId(caller), body));
        record(caller, note.getId(), ActivityKind.ACTIVITY_KIND_COMMENTED, BlockCodec.preview(body));

        return AddCommentResponse.newBuilder().setComment(NotesMapper.toProto(comment)).build();
    }

    ListCommentsResponse listComments(ListCommentsRequest request) {
        UUID familyId = callers.requireFamilyId();
        Long noteId = id(request.getNoteId(), "note_id");

        ListCommentsResponse.Builder response = ListCommentsResponse.newBuilder();
        comments.findForNote(familyId, noteId, request.getIncludeResolved())
                .forEach(comment -> response.addComments(NotesMapper.toProto(comment)));
        return response.build();
    }

    ResolveCommentResponse resolveComment(ResolveCommentRequest request) {
        Caller caller = callers.require();
        UUID userId = requireUserId(caller);

        NoteComment comment = comments.findByIdAndFamilyId(id(request.getCommentId(), "comment_id"), caller.familyId())
                .orElseThrow(() -> ConnectException.notFound("comment not found"));
        Note note = notes.findByIdAndFamilyId(comment.getNoteId(), caller.familyId())
                .orElseThrow(() -> ConnectException.notFound("comment not found"));

        if (!userId.equals(comment.getAuthorUserId()) && !userId.equals(note.getOwnerUserId())) {
            throw ConnectException.notFound("comment not found");
        }

        comment.resolve(request.getResolved());
        return ResolveCommentResponse.newBuilder().setComment(NotesMapper.toProto(comment)).build();
    }

    ListActivityResponse listActivity(ListActivityRequest request) {
        UUID familyId = callers.requireFamilyId();
        Long noteId = id(request.getNoteId(), "note_id");

        int limit = request.getLimit() > 0
                ? Math.min(request.getLimit(), MAX_ACTIVITY_LIMIT)
                : DEFAULT_ACTIVITY_LIMIT;

        ListActivityResponse.Builder response = ListActivityResponse.newBuilder();
        activity.findForNote(familyId, noteId, PageRequest.of(0, limit))
                .forEach(entry -> response.addActivity(NotesMapper.toProto(entry)));
        return response.build();
    }

    private void record(Caller caller, Long noteId, ActivityKind kind, String detail) {
        activity.save(NoteActivity.of(caller.familyId(), noteId, requireUserId(caller), kind.getNumber(), detail));
    }

    private static Specification<Note> filter(
            UUID familyId, Long notebookId, boolean starredOnly, boolean includeArchived, boolean archivedOnly) {
        return (root, query, builder) -> {
            List<Predicate> predicates = new ArrayList<>();
            predicates.add(builder.equal(root.get("familyId"), familyId));

            if (notebookId != null) {
                predicates.add(builder.equal(root.get("group").get("id"), notebookId));
            }
            if (starredOnly) {
                predicates.add(builder.isTrue(root.get("starred")));
            }
            if (archivedOnly) {
                predicates.add(builder.isTrue(root.get("archived")));
            } else if (!includeArchived) {
                predicates.add(builder.isFalse(root.get("archived")));
            }
            return builder.and(predicates.toArray(Predicate[]::new));
        };
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

    private static UUID requireUserId(Caller caller) {
        UUID userId = userId(caller);
        if (userId == null) {
            throw ConnectException.invalidArgument("the caller's subject is not a user id");
        }
        return userId;
    }

    private static UUID userId(Caller caller) {
        try {
            return UUID.fromString(caller.userId());
        } catch (IllegalArgumentException ex) {
            return null;
        }
    }
}
