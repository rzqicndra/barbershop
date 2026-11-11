import { useEffect, useState, type FormEvent } from 'react'
import { ImagePlus, Plus, Trash2 } from 'lucide-react'
import { api, messageOf } from './api'
import type { FAQ, GalleryPhoto, MediaAsset, StudioProps } from './data'
import { DeleteDialog, Modal } from './ui'

export function ImagePicker({
  label,
  value,
  onChange,
  onBusy,
  clearable = false,
}: {
  label: string
  value: string
  onChange: (url: string) => void
  onBusy?: (delta: number) => void
  clearable?: boolean
}) {
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function upload(file?: File) {
    if (!file || pending) return
    if (file.size > 8 * 1024 * 1024) {
      setError('Ukuran foto maksimal 8 MB.')
      return
    }
    setPending(true)
    setError('')
    onBusy?.(1)
    try {
      const form = new FormData()
      form.append('file', file)
      const result = await api<MediaAsset>('/admin/media', { method: 'POST', body: form })
      onChange(result.url)
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
      onBusy?.(-1)
    }
  }
  return (
    <div className="image-picker">
      <span className="field-label">{label}</span>
      <div className="image-picker-preview">
        {value ? (
          <img src={value} alt={'Pratinjau ' + label} />
        ) : (
          <div className="image-placeholder">
            <ImagePlus size={26} />
            <span>Belum ada foto</span>
          </div>
        )}
      </div>
      <label className="button button-outline upload-button">
        <ImagePlus size={16} />
        {pending ? 'Mengunggah…' : value ? 'Ganti foto' : 'Unggah foto'}
        <input
          type="file"
          aria-label={label + ' — unggah foto'}
          accept="image/jpeg,image/png,image/webp"
          disabled={pending}
          onChange={(event) => {
            void upload(event.target.files?.[0])
            event.target.value = ''
          }}
        />
      </label>
      {clearable && value && (
        <button
          type="button"
          className="text-link danger-text"
          disabled={pending}
          onClick={() => onChange('')}
        >
          Lepas gambar
        </button>
      )}
      <small>JPG, PNG, atau WebP, maksimal 8 MB. Simpan formulir setelah unggahan selesai.</small>
      {error && (
        <p className="field-error" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}

export function PublicPageSettings({ studio, refresh }: StudioProps) {
  const [draft, setDraft] = useState(studio.profile)
  const [uploads, setUploads] = useState(0)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  useEffect(() => setDraft(studio.profile), [studio.profile.version])
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (pending || uploads) return
    setPending(true)
    setMessage('')
    setError('')
    try {
      await api('/admin/profile', { method: 'PUT', body: draft })
      await refresh()
      setMessage('Tampilan publik tersimpan.')
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  const field = (key: keyof typeof draft, value: string) =>
    setDraft((current) => ({ ...current, [key]: value }))
  return (
    <form className="settings-panel" onSubmit={submit}>
      <h2>Tampilan halaman publik</h2>
      <p>Ubah identitas, foto pembuka, dan teks yang dilihat pelanggan.</p>
      <div className="form-pair">
        <label className="field">
          <span>Nama pada logo teks</span>
          <input
            required
            minLength={2}
            maxLength={30}
            value={draft.wordmark || ''}
            onChange={(e) => field('wordmark', e.target.value)}
          />
        </label>
        <label className="field">
          <span>Label pembuka</span>
          <input
            maxLength={80}
            value={draft.heroEyebrow || ''}
            onChange={(e) => field('heroEyebrow', e.target.value)}
          />
        </label>
      </div>
      <label className="field">
        <span>Judul utama</span>
        <textarea
          required
          minLength={2}
          maxLength={100}
          rows={2}
          value={draft.heroTitle || ''}
          onChange={(e) => field('heroTitle', e.target.value)}
        />
        <small>Gunakan Enter untuk memisahkan baris, maksimal tiga baris.</small>
      </label>
      <label className="field">
        <span>Deskripsi utama</span>
        <textarea
          required
          minLength={5}
          maxLength={500}
          rows={3}
          value={draft.heroDescription || ''}
          onChange={(e) => field('heroDescription', e.target.value)}
        />
      </label>
      <div className="image-settings-grid">
        <ImagePicker
          label="Foto pembuka"
          value={draft.heroImageUrl || '/assets/gilir-chair.webp'}
          onChange={(url) => field('heroImageUrl', url)}
          onBusy={(delta) => setUploads((n) => n + delta)}
        />
        <ImagePicker
          label="Logo gambar (opsional)"
          value={draft.logoImageUrl || ''}
          onChange={(url) => field('logoImageUrl', url)}
          onBusy={(delta) => setUploads((n) => n + delta)}
          clearable
        />
      </div>
      <div className="form-pair">
        <label className="field">
          <span>Judul galeri</span>
          <input
            required
            minLength={2}
            maxLength={100}
            value={draft.galleryTitle || ''}
            onChange={(e) => field('galleryTitle', e.target.value)}
          />
        </label>
        <label className="field">
          <span>Tagline footer</span>
          <input
            maxLength={120}
            value={draft.footerTagline || ''}
            onChange={(e) => field('footerTagline', e.target.value)}
          />
        </label>
      </div>
      <label className="field">
        <span>Deskripsi galeri</span>
        <textarea
          rows={2}
          maxLength={300}
          value={draft.galleryDescription || ''}
          onChange={(e) => field('galleryDescription', e.target.value)}
        />
      </label>
      {error && (
        <p className="notice notice-error" role="alert">
          {error}
        </p>
      )}
      {message && (
        <p className="notice notice-success" role="status">
          {message}
        </p>
      )}
      <button type="submit" className="button button-primary" disabled={pending || uploads > 0}>
        {pending ? 'Menyimpan…' : 'Simpan tampilan publik'}
      </button>
    </form>
  )
}

export function GallerySettings({ studio, refresh }: StudioProps) {
  const [editor, setEditor] = useState<GalleryPhoto | 'new' | null>(null)
  const [deleting, setDeleting] = useState<GalleryPhoto | null>(null)
  return (
    <section className="settings-panel">
      <div className="settings-section-head">
        <div>
          <h2>Galeri hasil karya</h2>
          <p>
            Unggah foto hasil potongan dengan izin pelanggan. Urutan terkecil tampil lebih dulu.
          </p>
        </div>
        <button className="button button-primary" type="button" onClick={() => setEditor('new')}>
          <Plus size={16} />
          Tambah foto
        </button>
      </div>
      {!(studio.gallery || []).length && (
        <p className="notice">
          Belum ada foto. Tambahkan foto asli barbershop untuk mengisi slide di halaman publik.
        </p>
      )}
      <div className="admin-gallery-grid">
        {(studio.gallery || []).map((photo) => (
          <article key={photo.id}>
            <img src={photo.imageUrl} alt={photo.altText} loading="lazy" />
            <div>
              <strong>{photo.title}</strong>
              <small>
                Urutan {photo.sortOrder} · {photo.active ? 'Tayang' : 'Draf'}
              </small>
              <div className="account-actions">
                <button
                  className="button button-outline"
                  type="button"
                  onClick={() => setEditor(photo)}
                >
                  Edit foto
                </button>
                <button
                  className="text-link danger-text"
                  type="button"
                  onClick={() => setDeleting(photo)}
                >
                  <Trash2 size={14} />
                  Hapus
                </button>
              </div>
            </div>
          </article>
        ))}
      </div>
      {editor && (
        <GalleryEditor
          photo={editor === 'new' ? undefined : editor}
          onClose={() => setEditor(null)}
          onSaved={refresh}
        />
      )}
      {deleting && (
        <DeleteDialog
          title={'Hapus foto ' + deleting.title + '?'}
          description="Foto akan dihapus dari galeri publik. Riwayat booking dan ulasan pelanggan tetap tersimpan."
          path={'/admin/gallery/' + deleting.id}
          version={deleting.version}
          onClose={() => setDeleting(null)}
          onDeleted={refresh}
        />
      )}
    </section>
  )
}
function GalleryEditor({
  photo,
  onClose,
  onSaved,
}: {
  photo?: GalleryPhoto
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const [draft, setDraft] = useState<GalleryPhoto>(
    photo || {
      id: '',
      imageUrl: '',
      title: '',
      caption: '',
      altText: '',
      sortOrder: 0,
      active: false,
      consentConfirmed: false,
      version: 0,
    },
  )
  const [uploads, setUploads] = useState(0)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (pending || uploads) return
    if (!draft.imageUrl) {
      setError('Unggah foto terlebih dahulu.')
      return
    }
    setPending(true)
    try {
      await api('/admin/gallery' + (photo ? '/' + photo.id : ''), {
        method: photo ? 'PUT' : 'POST',
        body: draft,
      })
      await onSaved()
      onClose()
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  return (
    <Modal title={photo ? 'Edit foto galeri' : 'Tambah foto galeri'} onClose={onClose} wide>
      <form className="dialog-form" onSubmit={submit}>
        <ImagePicker
          label="Foto hasil karya"
          value={draft.imageUrl}
          onChange={(url) => setDraft((current) => ({ ...current, imageUrl: url }))}
          onBusy={(delta) => setUploads((n) => n + delta)}
        />
        <div className="form-pair">
          <label className="field">
            <span>Judul foto</span>
            <input
              required
              minLength={2}
              maxLength={100}
              value={draft.title}
              onChange={(e) => setDraft({ ...draft, title: e.target.value })}
            />
          </label>
          <label className="field">
            <span>Urutan tampil</span>
            <input
              type="number"
              min={0}
              max={10000}
              required
              value={draft.sortOrder}
              onChange={(e) => setDraft({ ...draft, sortOrder: Number(e.target.value) })}
            />
          </label>
        </div>
        <label className="field">
          <span>Keterangan (opsional)</span>
          <input
            maxLength={300}
            value={draft.caption}
            onChange={(e) => setDraft({ ...draft, caption: e.target.value })}
          />
        </label>
        <label className="field">
          <span>Deskripsi foto untuk aksesibilitas</span>
          <input
            required
            minLength={5}
            maxLength={200}
            value={draft.altText}
            onChange={(e) => setDraft({ ...draft, altText: e.target.value })}
          />
          <small>Jelaskan hasil atau gaya rambut yang terlihat pada foto.</small>
        </label>
        <label className="check-field">
          <input
            type="checkbox"
            checked={draft.consentConfirmed}
            onChange={(e) => setDraft({ ...draft, consentConfirmed: e.target.checked })}
          />
          Saya memiliki hak menggunakan foto ini dan izin orang yang ditampilkan, jika diperlukan.
        </label>
        <label className="check-field">
          <input
            type="checkbox"
            checked={draft.active}
            onChange={(e) => setDraft({ ...draft, active: e.target.checked })}
          />
          Tampilkan di galeri publik
        </label>
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button className="button button-outline" type="button" onClick={onClose}>
            Kembali
          </button>
          <button className="button button-primary" type="submit" disabled={pending || uploads > 0}>
            {pending ? 'Menyimpan…' : 'Simpan foto'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

export function FAQSettings({ studio, refresh }: StudioProps) {
  const [editor, setEditor] = useState<FAQ | 'new' | null>(null)
  const [deleting, setDeleting] = useState<FAQ | null>(null)
  return (
    <section className="settings-panel">
      <div className="settings-section-head">
        <div>
          <h2>Pertanyaan pelanggan</h2>
          <p>Kelola pertanyaan dan jawaban pada bagian lokasi.</p>
        </div>
        <button type="button" className="button button-primary" onClick={() => setEditor('new')}>
          <Plus size={16} />
          Tambah FAQ
        </button>
      </div>
      <div className="settings-list">
        {(studio.faqs || []).map((faq) => (
          <div key={faq.id}>
            <span>
              <strong>{faq.question}</strong>
              <small>
                Urutan {faq.sortOrder} · {faq.active ? 'Tayang' : 'Nonaktif'}
              </small>
            </span>
            <div className="account-actions">
              <button
                className="button button-outline"
                type="button"
                onClick={() => setEditor(faq)}
              >
                Edit
              </button>
              <button
                className="text-link danger-text"
                type="button"
                onClick={() => setDeleting(faq)}
              >
                Hapus
              </button>
            </div>
          </div>
        ))}
      </div>
      {editor && (
        <FAQEditor
          faq={editor === 'new' ? undefined : editor}
          onClose={() => setEditor(null)}
          onSaved={refresh}
        />
      )}
      {deleting && (
        <DeleteDialog
          title="Hapus pertanyaan?"
          description={deleting.question}
          path={'/admin/faqs/' + deleting.id}
          version={deleting.version}
          onClose={() => setDeleting(null)}
          onDeleted={refresh}
        />
      )}
    </section>
  )
}
function FAQEditor({
  faq,
  onClose,
  onSaved,
}: {
  faq?: FAQ
  onClose: () => void
  onSaved: () => Promise<void>
}) {
  const [draft, setDraft] = useState<FAQ>(
    faq || { id: '', question: '', answer: '', sortOrder: 0, active: true, version: 0 },
  )
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (pending) return
    setPending(true)
    try {
      await api('/admin/faqs' + (faq ? '/' + faq.id : ''), {
        method: faq ? 'PUT' : 'POST',
        body: draft,
      })
      await onSaved()
      onClose()
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setPending(false)
    }
  }
  return (
    <Modal title={faq ? 'Edit FAQ' : 'Tambah FAQ'} onClose={onClose}>
      <form className="dialog-form" onSubmit={submit}>
        <label className="field">
          <span>Pertanyaan</span>
          <input
            required
            minLength={5}
            maxLength={150}
            value={draft.question}
            onChange={(e) => setDraft({ ...draft, question: e.target.value })}
          />
        </label>
        <label className="field">
          <span>Jawaban</span>
          <textarea
            required
            minLength={5}
            maxLength={1500}
            rows={4}
            value={draft.answer}
            onChange={(e) => setDraft({ ...draft, answer: e.target.value })}
          />
        </label>
        <label className="field">
          <span>Urutan tampil</span>
          <input
            type="number"
            min={0}
            max={10000}
            value={draft.sortOrder}
            onChange={(e) => setDraft({ ...draft, sortOrder: Number(e.target.value) })}
          />
        </label>
        <label className="check-field">
          <input
            type="checkbox"
            checked={draft.active}
            onChange={(e) => setDraft({ ...draft, active: e.target.checked })}
          />
          Tampilkan di halaman publik
        </label>
        {error && (
          <p className="notice notice-error" role="alert">
            {error}
          </p>
        )}
        <div className="dialog-actions">
          <button type="button" className="button button-outline" onClick={onClose}>
            Kembali
          </button>
          <button type="submit" className="button button-primary" disabled={pending}>
            Simpan FAQ
          </button>
        </div>
      </form>
    </Modal>
  )
}
