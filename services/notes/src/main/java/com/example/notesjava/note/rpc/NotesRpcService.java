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
import com.example.notesjava.image.ImageStore;
import com.example.notesjava.image.ImageUpload;
import com.example.notesjava.share.domain.Share;
import com.example.notesjava.share.repository.AccessSpecifications;
import com.example.notesjava.share.service.AccessService;
import com.example.notesjava.share.service.NoteAccess;
import com.example.notesjava.share.service.NotebookAccess;
import com.example.notesjava.share.service.ShareService;
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
import com.nnc.familymanager.notes.v1.ListSharedWithMeResponse;
import com.nnc.familymanager.notes.v1.ListSharesRequest;
import com.nnc.familymanager.notes.v1.ListSharesResponse;
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
import com.nnc.familymanager.notes.v1.ShareNoteResponse;
import com.nnc.familymanager.notes.v1.ShareNotebookRequest;
import com.nnc.familymanager.notes.v1.ShareNotebookResponse;
import com.nnc.familymanager.notes.v1.SharePermission;
import com.nnc.familymanager.notes.v1.ShareSubject;
import com.nnc.familymanager.notes.v1.ToggleStarRequest;
import com.nnc.familymanager.notes.v1.ToggleStarResponse;
import com.nnc.familymanager.notes.v1.UnshareRequest;
import com.nnc.familymanager.notes.v1.UnshareResponse;
import com.nnc.familymanager.notes.v1.UpdateNoteRequest;
import com.nnc.familymanager.notes.v1.UpdateNoteResponse;
import com.nnc.familymanager.notes.v1.UpdateNotebookRequest;
import com.nnc.familymanager.notes.v1.UpdateNotebookResponse;
import com.nnc.familymanager.notes.v1.UploadNoteImageRequest;
import com.nnc.familymanager.notes.v1.UploadNoteImageResponse;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Sort;
import org.springframework.data.jpa.domain.Specification;
import jakarta.persistence.criteria.Predicate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.support.TransactionTemplate;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.UUID;
import java.util.function.Function;

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
    private final AccessService access;
    private final ShareService sharing;
    private final ObjectProvider<ImageStore> images;
    private final CallerContext callers;
    private final TransactionTemplate write;
    private final TransactionTemplate read;

    public NotesRpcService(NoteRepository notes, GroupRepository groups, NoteCommentRepository comments,
                           NoteActivityRepository activity, AccessService access, ShareService sharing,
                           ObjectProvider<ImageStore> images, CallerContext callers,
                           PlatformTransactionManager transactions) {
        this.notes = notes;
        this.groups = groups;
        this.comments = comments;
        this.activity = activity;
        this.access = access;
        this.sharing = sharing;
        this.images = images;
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
        register("UploadNoteImage", UploadNoteImageRequest.getDefaultInstance(),
                inTransaction(read, this::uploadNoteImage));

        register("ShareNote", ShareNoteRequest.getDefaultInstance(), inTransaction(write, this::shareNote));
        register("ShareNotebook", ShareNotebookRequest.getDefaultInstance(),
                inTransaction(write, this::shareNotebook));
        register("Unshare", UnshareRequest.getDefaultInstance(), inTransaction(write, this::unshare));
        register("ListShares", ListSharesRequest.getDefaultInstance(), inTransaction(read, this::listShares));
        register("ListSharedWithMe", ListSharedWithMeRequest.getDefaultInstance(),
                inTransaction(read, this::listSharedWithMe));

        register("AddComment", AddCommentRequest.getDefaultInstance(), inTransaction(write, this::addComment));
        register("ListComments", ListCommentsRequest.getDefaultInstance(), inTransaction(read, this::listComments));
        register("ResolveComment", ResolveCommentRequest.getDefaultInstance(), inTransaction(write, this::resolveComment));
        register("ListActivity", ListActivityRequest.getDefaultInstance(), inTransaction(read, this::listActivity));
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

    ListNotebooksResponse listNotebooks(ListNotebooksRequest request) {
        Viewer viewer = viewer();
        ListNotebooksResponse.Builder response = ListNotebooksResponse.newBuilder();

        for (Group group : groups.findAll(AccessSpecifications.visibleNotebooks(viewer.familyId(), viewer.userId()))) {
            response.addNotebooks(NotesMapper.toProto(group, noteCount(viewer, group)));
        }
        return response.build();
    }

    CreateNotebookResponse createNotebook(CreateNotebookRequest request) {
        Caller caller = callers.require();
        String name = request.getName().trim();
        if (name.isEmpty()) {
            throw ConnectException.invalidArgument("name is required");
        }

        Group group = Group.of(caller.familyId(), name);
        group.owner(userId(caller));
        Group saved = groups.save(group);
        return CreateNotebookResponse.newBuilder()
                .setNotebook(NotesMapper.toProto(saved, 0))
                .build();
    }

    UpdateNotebookResponse updateNotebook(UpdateNotebookRequest request) {
        Viewer viewer = viewer();
        Group group = managedNotebook(request.getNotebookId(), viewer);

        String name = request.getName().trim();
        if (name.isEmpty()) {
            throw ConnectException.invalidArgument("name is required");
        }
        group.rename(name, group.getColor());

        return UpdateNotebookResponse.newBuilder()
                .setNotebook(NotesMapper.toProto(group, noteCount(viewer, group)))
                .build();
    }

    DeleteNotebookResponse deleteNotebook(DeleteNotebookRequest request) {
        Viewer viewer = viewer();
        Group group = managedNotebook(request.getNotebookId(), viewer);

        notes.clearGroup(group.getId(), viewer.familyId());
        groups.delete(group);
        return DeleteNotebookResponse.getDefaultInstance();
    }

    ListNotesResponse listNotes(ListNotesRequest request) {
        Viewer viewer = viewer();

        int pageSize = pageSize(request.getPageSize());
        Long notebookId = request.getNotebookId().isBlank() ? null : id(request.getNotebookId(), "notebook_id");

        Specification<Note> spec = AccessSpecifications.visibleNotes(viewer.familyId(), viewer.userId())
                .and(filter(viewer.familyId(), notebookId, request.getStarredOnly(),
                        request.getIncludeArchived(), request.getArchivedOnly()));
        if (request.getSharedOnly()) {
            spec = spec.and(AccessSpecifications.sharedWithMe(viewer.familyId(), viewer.userId()));
        }

        Page<Note> page = notes.findAll(spec, PageRequest.of(0, pageSize, sortFor(request.getSort())));

        ListNotesResponse.Builder response = ListNotesResponse.newBuilder();
        response.addAllNotes(toProto(viewer, page.getContent()));
        return response.build();
    }

    GetNoteResponse getNote(GetNoteRequest request) {
        Seen seen = visibleNote(request.getNoteId(), viewer());
        return GetNoteResponse.newBuilder()
                .setNote(NotesMapper.toProto(seen.note(), seen.access()))
                .build();
    }

    CreateNoteResponse createNote(CreateNoteRequest request) {
        Caller caller = callers.require();
        Viewer viewer = viewer(caller);

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
                : fileableNotebook(request.getNotebookId(), viewer);

        Note note = Note.of(viewer.familyId(), title, BlockCodec.toPlainText(blocks), group);
        note.owner(viewer.userId());
        note.writeBlocks(BlockCodec.encode(blocks), BlockCodec.toPlainText(blocks));
        Note saved = notes.save(note);
        record(caller, saved.getId(), ActivityKind.ACTIVITY_KIND_CREATED, "");

        return CreateNoteResponse.newBuilder()
                .setNote(NotesMapper.toProto(saved, access.note(viewer.familyId(), viewer.userId(), saved)))
                .build();
    }

    UpdateNoteResponse updateNote(UpdateNoteRequest request) {
        Caller caller = callers.require();
        Seen seen = editableNote(request.getNoteId(), viewer(caller));
        Note note = seen.note();

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

        return UpdateNoteResponse.newBuilder().setNote(NotesMapper.toProto(note, seen.access())).build();
    }

    MoveNoteResponse moveNote(MoveNoteRequest request) {
        Viewer viewer = viewer();
        Seen seen = managedNote(request.getNoteId(), viewer);
        Note note = seen.note();

        note.moveTo(request.getNotebookId().isBlank() ? null : fileableNotebook(request.getNotebookId(), viewer));
        return MoveNoteResponse.newBuilder()
                .setNote(NotesMapper.toProto(note, access.note(viewer.familyId(), viewer.userId(), note)))
                .build();
    }

    ToggleStarResponse toggleStar(ToggleStarRequest request) {
        Seen seen = editableNote(request.getNoteId(), viewer());

        seen.note().toggleStar();
        return ToggleStarResponse.newBuilder().setNote(NotesMapper.toProto(seen.note(), seen.access())).build();
    }

    ArchiveNoteResponse archiveNote(ArchiveNoteRequest request) {
        Seen seen = editableNote(request.getNoteId(), viewer());

        seen.note().archive(request.getArchived());
        return ArchiveNoteResponse.newBuilder().setNote(NotesMapper.toProto(seen.note(), seen.access())).build();
    }

    DeleteNoteResponse deleteNote(DeleteNoteRequest request) {
        notes.delete(managedNote(request.getNoteId(), viewer()).note());
        return DeleteNoteResponse.getDefaultInstance();
    }

    UploadNoteImageResponse uploadNoteImage(UploadNoteImageRequest request) {
        ImageStore store = images.getIfAvailable();
        if (store == null) {
            throw ConnectException.unavailable("image storage is not configured on this server");
        }

        ImageUpload.Accepted accepted;
        byte[] data = request.getImage().toByteArray();
        try {
            accepted = ImageUpload.validate(request.getContentType(), data);
        } catch (IllegalArgumentException ex) {
            throw ConnectException.invalidArgument(ex.getMessage());
        }

        Viewer viewer = viewer();
        Note note = editableNote(request.getNoteId(), viewer).note();

        String key = ImageUpload.key(viewer.familyId(), note.getId(), data, accepted.extension());
        String url = store.put(key, data, accepted.contentType());
        return UploadNoteImageResponse.newBuilder().setImageUrl(url).build();
    }

    SearchResponse search(SearchRequest request) {
        Viewer viewer = viewer();

        String query = request.getQuery().trim();
        if (query.isEmpty()) {
            throw ConnectException.invalidArgument("query is required");
        }

        long started = System.nanoTime();
        int limit = request.getLimit() > 0 ? Math.min(request.getLimit(), MAX_PAGE_SIZE) : DEFAULT_SEARCH_LIMIT;
        String needle = "%" + query.toLowerCase(Locale.ROOT) + "%";

        SearchResponse.Builder response = SearchResponse.newBuilder();
        List<Note> hits = notes.findAll(
                AccessSpecifications.visibleNotes(viewer.familyId(), viewer.userId()).and(matching(needle)),
                PageRequest.of(0, limit, Sort.by(Sort.Direction.DESC, "updatedAt"))).getContent();

        Map<Long, Boolean> notebookVisible = new HashMap<>();
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
                Group group = note.getGroup();
                hit.setNotebookId(String.valueOf(group.getId()));
                if (notebookVisible.computeIfAbsent(group.getId(),
                        id -> access.notebook(viewer.familyId(), viewer.userId(), group).visible())) {
                    hit.setContext(group.getTitle());
                }
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

    ShareNoteResponse shareNote(ShareNoteRequest request) {
        Caller caller = callers.require();
        UUID me = requireUserId(caller);
        UUID member = shareMember(request.getSubject(), request.getMemberUserId(), me);
        short permission = permission(request.getPermission());

        Note note = visibleNote(request.getNoteId(), viewer(caller)).note();
        if (!me.equals(note.getOwnerUserId())) {
            throw ConnectException.permissionDenied("only the note's owner can share it");
        }

        Share share = sharing.shareNote(caller.familyId(), note.getId(), member, permission, me);
        record(caller, note.getId(), ActivityKind.ACTIVITY_KIND_SHARED, member == null ? "" : member.toString());
        return ShareNoteResponse.newBuilder().setShare(NotesMapper.toProto(share)).build();
    }

    ShareNotebookResponse shareNotebook(ShareNotebookRequest request) {
        Caller caller = callers.require();
        UUID me = requireUserId(caller);
        UUID member = shareMember(request.getSubject(), request.getMemberUserId(), me);
        short permission = permission(request.getPermission());

        Group group = visibleNotebook(request.getNotebookId(), viewer(caller));
        if (!me.equals(group.getOwnerUserId())) {
            throw ConnectException.permissionDenied("only the notebook's owner can share it");
        }

        Share share = sharing.shareNotebook(caller.familyId(), group.getId(), member, permission, me);
        String detail = member == null ? "" : member.toString();
        for (Note note : notes.findByFamilyIdAndGroup_IdAndOwnerUserId(caller.familyId(), group.getId(), me)) {
            record(caller, note.getId(), ActivityKind.ACTIVITY_KIND_SHARED, detail);
        }
        return ShareNotebookResponse.newBuilder().setShare(NotesMapper.toProto(share)).build();
    }

    UnshareResponse unshare(UnshareRequest request) {
        Viewer viewer = viewer();
        Long shareId = id(request.getShareId(), "share_id");
        boolean onNote = exactlyOneTarget(request.getNoteId(), request.getNotebookId());

        boolean revoked;
        if (onNote) {
            Note note = notes.findByIdAndFamilyId(id(request.getNoteId(), "note_id"), viewer.familyId())
                    .filter(found -> viewer.userId() != null && viewer.userId().equals(found.getOwnerUserId()))
                    .orElseThrow(() -> ConnectException.notFound("share not found"));
            revoked = sharing.revokeFromNote(viewer.familyId(), note.getId(), shareId);
        } else {
            Group group = groups.findByIdAndFamilyId(id(request.getNotebookId(), "notebook_id"), viewer.familyId())
                    .filter(found -> viewer.userId() != null && viewer.userId().equals(found.getOwnerUserId()))
                    .orElseThrow(() -> ConnectException.notFound("share not found"));
            revoked = sharing.revokeFromNotebook(viewer.familyId(), group.getId(), shareId);
        }
        if (!revoked) {
            throw ConnectException.notFound("share not found");
        }
        return UnshareResponse.getDefaultInstance();
    }

    ListSharesResponse listShares(ListSharesRequest request) {
        Viewer viewer = viewer();
        boolean onNote = exactlyOneTarget(request.getNoteId(), request.getNotebookId());

        List<Share> shares = onNote
                ? sharing.onNote(viewer.familyId(), visibleNote(request.getNoteId(), viewer).note().getId())
                : sharing.onNotebook(viewer.familyId(), visibleNotebook(request.getNotebookId(), viewer).getId());

        ListSharesResponse.Builder response = ListSharesResponse.newBuilder();
        shares.forEach(share -> response.addShares(NotesMapper.toProto(share)));
        return response.build();
    }

    ListSharedWithMeResponse listSharedWithMe(ListSharedWithMeRequest request) {
        Viewer viewer = viewer();
        int pageSize = pageSize(request.getPageSize());

        List<Note> shared = notes.findAll(
                AccessSpecifications.sharedWithMe(viewer.familyId(), viewer.userId())
                        .and((root, query, builder) -> builder.isFalse(root.get("archived"))),
                PageRequest.of(0, pageSize, Sort.by(Sort.Direction.DESC, "updatedAt"))).getContent();

        ListSharedWithMeResponse.Builder response = ListSharedWithMeResponse.newBuilder()
                .addAllNotes(toProto(viewer, shared));
        for (Group group : groups.findAll(
                AccessSpecifications.notebooksSharedWithMe(viewer.familyId(), viewer.userId()),
                Sort.by(Sort.Direction.ASC, "title"))) {
            response.addNotebooks(NotesMapper.toProto(group, noteCount(viewer, group)));
        }
        return response.build();
    }

    AddCommentResponse addComment(AddCommentRequest request) {
        Caller caller = callers.require();
        Note note = visibleNote(request.getNoteId(), viewer(caller)).note();

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
        Viewer viewer = viewer();
        Long noteId = id(request.getNoteId(), "note_id");

        ListCommentsResponse.Builder response = ListCommentsResponse.newBuilder();
        if (!canSee(viewer, noteId)) {
            return response.build();
        }
        comments.findForNote(viewer.familyId(), noteId, request.getIncludeResolved())
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

        if (!access.note(caller.familyId(), userId, note).visible()) {
            throw ConnectException.notFound("comment not found");
        }
        if (!userId.equals(comment.getAuthorUserId()) && !userId.equals(note.getOwnerUserId())) {
            throw ConnectException.notFound("comment not found");
        }

        comment.resolve(request.getResolved());
        return ResolveCommentResponse.newBuilder().setComment(NotesMapper.toProto(comment)).build();
    }

    ListActivityResponse listActivity(ListActivityRequest request) {
        Viewer viewer = viewer();
        Long noteId = id(request.getNoteId(), "note_id");

        int limit = request.getLimit() > 0
                ? Math.min(request.getLimit(), MAX_ACTIVITY_LIMIT)
                : DEFAULT_ACTIVITY_LIMIT;

        ListActivityResponse.Builder response = ListActivityResponse.newBuilder();
        if (!canSee(viewer, noteId)) {
            return response.build();
        }
        activity.findForNote(viewer.familyId(), noteId, PageRequest.of(0, limit))
                .forEach(entry -> response.addActivity(NotesMapper.toProto(entry)));
        return response.build();
    }

    private void record(Caller caller, Long noteId, ActivityKind kind, String detail) {
        activity.save(NoteActivity.of(caller.familyId(), noteId, requireUserId(caller), kind.getNumber(), detail));
    }

    private record Viewer(UUID familyId, UUID userId) {
    }

    private record Seen(Note note, NoteAccess access) {
    }

    private Viewer viewer() {
        return viewer(callers.require());
    }

    private static Viewer viewer(Caller caller) {
        return new Viewer(caller.familyId(), userId(caller));
    }

    private List<com.nnc.familymanager.notes.v1.Note> toProto(Viewer viewer, List<Note> page) {
        Map<Long, NoteAccess> grants = access.notes(viewer.familyId(), viewer.userId(), page);
        List<com.nnc.familymanager.notes.v1.Note> out = new ArrayList<>(page.size());
        for (Note note : page) {
            out.add(NotesMapper.toProto(note, grants.get(note.getId())));
        }
        return out;
    }

    private long noteCount(Viewer viewer, Group group) {
        Long groupId = group.getId();
        return notes.count(AccessSpecifications.visibleNotes(viewer.familyId(), viewer.userId())
                .and((root, query, builder) -> builder.equal(root.get("group").get("id"), groupId)));
    }

    private boolean canSee(Viewer viewer, Long noteId) {
        return notes.findWithRelationsByIdAndFamilyId(noteId, viewer.familyId())
                .map(note -> access.note(viewer.familyId(), viewer.userId(), note).visible())
                .orElse(false);
    }

    private Seen visibleNote(String noteId, Viewer viewer) {
        Note note = notes.findWithRelationsByIdAndFamilyId(id(noteId, "note_id"), viewer.familyId())
                .orElseThrow(() -> ConnectException.notFound("that note does not exist"));
        NoteAccess grant = access.note(viewer.familyId(), viewer.userId(), note);
        if (!grant.visible()) {
            throw ConnectException.notFound("that note does not exist");
        }
        return new Seen(note, grant);
    }

    private Seen editableNote(String noteId, Viewer viewer) {
        Seen seen = visibleNote(noteId, viewer);
        if (!seen.access().canEdit()) {
            throw ConnectException.permissionDenied("this note is shared with you to view, not to edit");
        }
        return seen;
    }

    private Seen managedNote(String noteId, Viewer viewer) {
        Seen seen = visibleNote(noteId, viewer);
        if (!seen.access().canManage()) {
            throw ConnectException.permissionDenied("only the note's owner can do that");
        }
        return seen;
    }

    private Group visibleNotebook(String notebookId, Viewer viewer) {
        return notebookWith(notebookId, viewer, grant -> true, "");
    }

    private Group fileableNotebook(String notebookId, Viewer viewer) {
        return notebookWith(notebookId, viewer, NotebookAccess::canFile,
                "this notebook is shared with you to view, not to add notes to");
    }

    private Group managedNotebook(String notebookId, Viewer viewer) {
        return notebookWith(notebookId, viewer, NotebookAccess::canManage, "only the notebook's owner can do that");
    }

    private Group notebookWith(String notebookId, Viewer viewer,
                               Function<NotebookAccess, Boolean> allowed, String refusal) {
        Group group = groups.findByIdAndFamilyId(id(notebookId, "notebook_id"), viewer.familyId())
                .orElseThrow(() -> ConnectException.notFound("that notebook does not exist"));
        NotebookAccess grant = access.notebook(viewer.familyId(), viewer.userId(), group);
        if (!grant.visible()) {
            throw ConnectException.notFound("that notebook does not exist");
        }
        if (!allowed.apply(grant)) {
            throw ConnectException.permissionDenied(refusal);
        }
        return group;
    }

    private static UUID shareMember(ShareSubject subject, String memberUserId, UUID me) {
        return switch (subject) {
            case SHARE_SUBJECT_MEMBER -> {
                UUID member;
                try {
                    member = UUID.fromString(memberUserId.trim());
                } catch (IllegalArgumentException ex) {
                    throw ConnectException.invalidArgument("member_user_id is required when subject is MEMBER");
                }
                if (member.equals(me)) {
                    throw ConnectException.invalidArgument("you already own this");
                }
                yield member;
            }
            case SHARE_SUBJECT_FAMILY -> {
                if (!memberUserId.isBlank()) {
                    throw ConnectException.invalidArgument("member_user_id must be empty when subject is FAMILY");
                }
                yield null;
            }
            default -> throw ConnectException.invalidArgument("subject must be MEMBER or FAMILY");
        };
    }

    private static short permission(SharePermission permission) {
        return permission == SharePermission.SHARE_PERMISSION_EDIT ? Share.EDIT : Share.VIEW;
    }

    private static boolean exactlyOneTarget(String noteId, String notebookId) {
        boolean note = !noteId.isBlank();
        boolean notebook = !notebookId.isBlank();
        if (note == notebook) {
            throw ConnectException.invalidArgument("exactly one of note_id and notebook_id is required");
        }
        return note;
    }

    private static int pageSize(int requested) {
        return requested > 0 ? Math.min(requested, MAX_PAGE_SIZE) : DEFAULT_PAGE_SIZE;
    }

    private static Specification<Note> matching(String needle) {
        return (root, query, builder) -> builder.and(
                builder.isFalse(root.get("archived")),
                builder.or(
                        builder.like(builder.lower(root.get("title")), needle),
                        builder.like(builder.lower(builder.coalesce(root.get("content"), "")), needle)));
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
