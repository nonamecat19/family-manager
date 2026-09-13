package com.example.notesjava.note.api;

import com.example.notesjava.common.web.PageResponse;
import com.example.notesjava.note.api.dto.CreateNoteRequest;
import com.example.notesjava.note.api.dto.NoteResponse;
import com.example.notesjava.note.api.dto.UpdateNoteRequest;
import com.example.notesjava.note.domain.NoteStatus;
import com.example.notesjava.note.service.NoteService;
import jakarta.validation.Valid;
import org.springframework.data.domain.Pageable;
import org.springframework.data.domain.Sort;
import org.springframework.data.web.PageableDefault;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.util.UriComponentsBuilder;

@RestController
@RequestMapping("/api/notes")
public class NoteController {

    private final NoteService noteService;

    public NoteController(NoteService noteService) {
        this.noteService = noteService;
    }

    @GetMapping
    public PageResponse<NoteResponse> list(
            @RequestParam(required = false) Long groupId,
            @RequestParam(required = false) NoteStatus status,
            @PageableDefault(size = 20, sort = "updatedAt", direction = Sort.Direction.DESC) Pageable pageable
    ) {
        return PageResponse.of(noteService.list(groupId, status, pageable));
    }

    @GetMapping("/{id}")
    public NoteResponse getById(@PathVariable Long id) {
        return noteService.getById(id);
    }

    @PostMapping
    public ResponseEntity<NoteResponse> create(
            @Valid @RequestBody CreateNoteRequest request,
            UriComponentsBuilder uriBuilder
    ) {
        NoteResponse created = noteService.create(request);
        return ResponseEntity
                .created(uriBuilder.path("/api/notes/{id}").build(created.id()))
                .body(created);
    }

    @PutMapping("/{id}")
    public NoteResponse update(@PathVariable Long id, @Valid @RequestBody UpdateNoteRequest request) {
        return noteService.update(id, request);
    }

    @DeleteMapping("/{id}")
    @ResponseStatus(HttpStatus.NO_CONTENT)
    public void delete(@PathVariable Long id) {
        noteService.delete(id);
    }
}
