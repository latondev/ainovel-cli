Tạo cấu trúc thể loại mới cho ainovel-cli dựa trên tên thể loại được truyền vào: $ARGUMENTS

## Việc cần làm

Tạo các file và folder sau trong repo `D:\YTB\New\ainovel-cli` (hoặc thư mục làm việc hiện tại nếu khác).

Thay `<genre>` bằng tên thể loại từ $ARGUMENTS (viết thường, không dấu, dùng dấu gạch ngang nếu nhiều từ — ví dụ: `horror`, `sci-fi`, `historical`).

---

### 1. `assets/references/genres/<genre>/style-references.md`

```markdown
# Tài liệu tham khảo phong cách — <genre>

<!-- Mô tả đặc trưng văn phong, giọng kể, cách dùng từ của thể loại này -->
<!-- Writer đọc ở 3 chương đầu để nắm tone -->

## Đặc trưng văn phong

(Viết vào đây)

## Từ ngữ đặc trưng

(Viết vào đây)

## Những gì cần tránh

(Viết vào đây)
```

---

### 2. `assets/references/genres/<genre>/arc-templates.md`

```markdown
# Mẫu cung truyện — <genre>

<!-- Architect đọc khi lên đề cương: các dạng cung truyện phổ biến của thể loại -->

## Cung truyện phổ biến

(Viết vào đây)

## Cấu trúc leo thang căng thẳng

(Viết vào đây)

## Điểm móc đặc trưng của thể loại

(Viết vào đây)
```

---

### 3. `assets/references/genres/<genre>/knowledge/lore.md`

```markdown
# Lore & Quy tắc thế giới — <genre>

<!-- Kiến thức nền: luật lệ, hệ thống, logic vận hành của thể loại -->
<!-- Writer đọc mọi chương để bám đúng logic -->

## Quy tắc cốt lõi

(Nghiên cứu và viết vào đây)

## Hệ thống / Cơ chế

(Viết vào đây)

## Ranh giới không được vi phạm

(Viết vào đây)
```

---

### 4. `assets/references/genres/<genre>/knowledge/psychology.md`

```markdown
# Tâm lý nhân vật — <genre>

<!-- Đặc điểm tâm lý điển hình của các kiểu nhân vật trong thể loại này -->
<!-- Giúp writer viết động cơ và phản ứng chân thực -->

## Tâm lý nhân vật chính

(Viết vào đây)

## Tâm lý phản diện / mối đe dọa

(Viết vào đây)

## Phản ứng cảm xúc đặc trưng

(Viết vào đây)
```

---

### 5. `assets/references/genres/<genre>/knowledge/authenticity.md`

```markdown
# Chi tiết thực tế — <genre>

<!-- Research notes: chi tiết cụ thể giúp truyện có độ chân thực cao -->
<!-- Nghi lễ, thủ tục, thuật ngữ, chi tiết kỹ thuật... -->

## Chi tiết đặc trưng cần bám sát

(Nghiên cứu và viết vào đây)

## Thuật ngữ / Từ chuyên ngành

(Viết vào đây)

## Những lỗi thường gặp cần tránh

(Viết vào đây)
```

---

### 6. `assets/styles/<genre>.md`

```markdown
## Phong cách viết — <genre>

<!-- Ghép vào system prompt của writer. Ngắn gọn, có thể thực thi ngay -->

- **Nhịp truyện**: (Viết đặc trưng nhịp của thể loại)
- **Miêu tả**: (Ưu tiên giác quan nào, mức độ chi tiết)
- **Đối thoại**: (Tone, mức độ ngầm ý)
- **Cảm xúc**: (Cách thể hiện cảm xúc đặc trưng của thể loại)
- **Bầu không khí**: (Cách dựng atmosphere)
```

---

## Sau khi tạo xong

Thông báo cho người dùng:
- Danh sách file đã tạo
- Nhắc rằng `knowledge/` có thể thêm bất kỳ file `.md` nào mới mà không cần sửa code
- Nhắc điền nội dung nghiên cứu vào các placeholder `(Viết vào đây)`
- Để dùng thể loại này: truyền `style: <genre>` khi tạo dự án
