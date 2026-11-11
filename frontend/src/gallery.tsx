import { useEffect, useRef, useState } from 'react'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import type { GalleryPhoto } from './data'

export function PhotoGallery({ photos, title }: { photos: GalleryPhoto[]; title: string }) {
  const track = useRef<HTMLDivElement>(null)
  const [edges, setEdges] = useState({ first: true, last: photos.length < 2 })
  function measure() {
    const element = track.current
    if (element)
      setEdges({
        first: element.scrollLeft < 4,
        last: element.scrollLeft + element.clientWidth >= element.scrollWidth - 4,
      })
  }
  function slide(direction: number) {
    track.current?.scrollBy({
      left: direction * track.current.clientWidth * 0.85,
      behavior: window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
        ? 'instant'
        : 'smooth',
    })
  }
  useEffect(() => {
    measure()
    window.addEventListener('resize', measure)
    return () => window.removeEventListener('resize', measure)
  }, [photos.length])
  if (!photos.length)
    return (
      <div className="gallery-empty">
        <strong>Hasil karya segera hadir.</strong>
        <p>Foto potongan dan perawatan akan ditampilkan di sini.</p>
      </div>
    )
  return (
    <section className="gallery-carousel" aria-label={title} aria-roledescription="carousel">
      <div className="gallery-controls">
        <span className="muted">{photos.length} foto hasil karya</span>
        <div>
          <button
            className="icon-button"
            type="button"
            aria-label="Foto sebelumnya"
            disabled={edges.first}
            onClick={() => slide(-1)}
          >
            <ArrowLeft size={19} />
          </button>
          <button
            className="icon-button"
            type="button"
            aria-label="Foto berikutnya"
            disabled={edges.last}
            onClick={() => slide(1)}
          >
            <ArrowRight size={19} />
          </button>
        </div>
      </div>
      <div
        className="gallery-track"
        ref={track}
        onScroll={measure}
        onLoad={measure}
        tabIndex={0}
        aria-label="Geser untuk melihat hasil karya"
      >
        {photos.map((photo) => (
          <figure className="gallery-card" key={photo.id}>
            <img src={photo.imageUrl} alt={photo.altText} loading="lazy" width={720} height={900} />
            <figcaption>
              <strong>{photo.title}</strong>
              {photo.caption && <span>{photo.caption}</span>}
            </figcaption>
          </figure>
        ))}
      </div>
    </section>
  )
}
