// Run through TestBrowserAcceptance: the Go harness owns the isolated database.
// The HTTPS self-signed certificate is trusted only in this local test context.
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const { createRequire } = require('node:module')
const { chromium } = createRequire(path.resolve('frontend/package.json'))('playwright-core')

async function main() {
  const base = process.env.E2E_URL
  assert(base && /^https:\/\/127\.0\.0\.1:\d+$/.test(base), 'Use the local Go test harness')
  assert(process.env.E2E_CONTROL_TOKEN, 'Missing isolated fixture control token')
  const output = process.env.E2E_ARTIFACT_DIR
  if (output) fs.mkdirSync(output, { recursive: true, mode: 0o700 })
  const browser = await chromium.launch({
    executablePath: process.env.CHROMIUM_EXECUTABLE || '/usr/bin/chromium',
    headless: true,
    args: ['--no-sandbox', '--no-zygote', '--disable-dev-shm-usage'],
  })
  const errors = []
  const failures = []
  const steps = []
  function watch(page) {
    page.on('pageerror', (error) => errors.push(error.message))
    page.on('console', (message) => {
      if (message.type() === 'error' && !message.location().url.includes('/auth/session') && !message.text().includes('ERR_ABORTED')) errors.push(message.text().replace(/data:[^'"\s]+/g,'data:…'))
    })
    page.on('response', (response) => {
      if (response.url().startsWith(base) && response.status() >= 500) failures.push(response.status() + ' ' + new URL(response.url()).pathname)
    })
  }
  async function has(locator) { await locator.waitFor({state:'visible',timeout:20000}) }
  try {
    const customer = await browser.newContext({ ignoreHTTPSErrors: true, viewport: {width:1440,height:1000} })
    const page = await customer.newPage();watch(page)
    await page.goto(base + '/#booking')
    await has(page.getByRole('heading',{name:'Pilih layanan.'}))
    await page.locator('.service-choice').filter({has:page.getByText('Potong rambut',{exact:true})}).click()
    await page.getByRole('button',{name:'Lanjut pilih jam'}).click()
    await has(page.getByRole('heading',{name:'Pilih barber & waktu.'}))
    await page.locator('.slot-grid button').first().click()
    await page.getByRole('button',{name:'Lanjut konfirmasi'}).click()
    await page.getByLabel('Nama',{exact:true}).fill('Pelanggan Uji Browser')
    await page.getByLabel('Nomor WhatsApp',{exact:true}).fill('081200000000')
    const created = page.waitForResponse((r)=>r.url()===base+'/api/v1/bookings' && r.request().method()==='POST')
    await page.getByRole('button',{name:'Konfirmasi booking',exact:true}).click()
    const response = await created
    assert.equal(response.status(),201)
    const booking = await response.json()
    await has(page.getByRole('heading',{name:'Jadwalmu sudah dicatat.'}))
    const privateURL = base+'/#booking/'+booking.token
    assert(await page.getByRole('button',{name:'Beri ulasan setelah selesai'}).isDisabled())
    const forbiddenReview=await customer.request.post(base+'/api/v1/bookings/current/review',{
      headers:{Authorization:'Bearer '+booking.token},data:{rating:5,displayName:'Pengunjung uji',comment:'Belum selesai'},
    })
    assert.equal(forbiddenReview.status(),409)
    steps.push('Customer booking and unfinished review protection')

    const adminContext=await browser.newContext({ignoreHTTPSErrors:true,viewport:{width:1440,height:1000}})
    const admin=await adminContext.newPage();watch(admin)
    await admin.goto(base+'/admin')
    await admin.getByLabel('Email staf').fill('owner@gilir.test')
    await admin.getByLabel('Kata sandi',{exact:true}).fill('Owner-password-2026')
    await admin.getByRole('button',{name:'Masuk',exact:true}).click()
    await has(admin.getByRole('heading',{name:'Agenda kunjungan'}))
    const sessionCookie=(await adminContext.cookies()).find((c)=>c.name==='gilir_session')
    assert(sessionCookie && sessionCookie.httpOnly && sessionCookie.secure && sessionCookie.sameSite==='Lax')
    await admin.goto(base+'/')
    await has(admin.getByRole('heading',{name:'Agenda kunjungan'}))
    assert.equal(new URL(admin.url()).pathname,'/admin/agenda')
    await admin.getByRole('button',{name:'Detail Pelanggan Uji Browser'}).click()
    assert(await admin.getByRole('button',{name:'Mulai layanan',exact:true}).isDisabled())
    await admin.getByRole('button',{name:'Tutup',exact:true}).click()
    const advance=await customer.request.post(base+'/__test/advance?bookingId='+booking.id,{headers:{'X-Test-Control':process.env.E2E_CONTROL_TOKEN}})
    assert.equal(advance.status(),204)
    await admin.reload()
    await admin.getByRole('button',{name:'Detail Pelanggan Uji Browser'}).click()
    await admin.getByRole('button',{name:'Mulai layanan',exact:true}).click()
    await admin.getByRole('button',{name:'Tandai selesai',exact:true}).click()
    await has(admin.getByRole('dialog').getByText('Selesai',{exact:true}))
    steps.push('Secure staff cookie, root redirect, timed service and completion')

    await page.goto(privateURL)
    await page.reload()
    await page.getByRole('radio',{name:'5 bintang',exact:true}).check()
    await page.getByLabel(/^Nama tampilan/).fill('Pengunjung Uji')
    await page.getByLabel('Ulasan (opsional)').fill('Ulasan untuk pengujian lokal otomatis.')
    await page.getByRole('button',{name:'Kirim ulasan',exact:true}).click()
    await has(page.getByText('Terima kasih untuk ulasanmu.'))
    assert.equal(await page.getByRole('radio').count(),0)
    steps.push('Completed visit review saved and duplicate form removed')

    await admin.goto(base+'/admin/pengaturan')
    await admin.getByRole('button',{name:'Galeri',exact:true}).click()
    await admin.getByRole('button',{name:'Tambah foto',exact:true}).click()
    await admin.getByLabel(/Foto hasil karya — unggah foto/).setInputFiles(path.resolve('frontend/public/assets/gilir-chair.webp'))
    await admin.getByLabel('Judul foto',{exact:true}).fill('Uji galeri lokal')
    await admin.getByLabel(/^Deskripsi foto untuk aksesibilitas/).fill('Ilustrasi kursi untuk pengujian galeri')
    await admin.getByRole('checkbox',{name:/Saya memiliki hak menggunakan foto/}).check()
    await admin.getByRole('checkbox',{name:'Tampilkan di galeri publik',exact:true}).check()
    await admin.getByRole('button',{name:'Simpan foto',exact:true}).click()
    await admin.getByRole('dialog').waitFor({state:'hidden'})
    await has(admin.getByText('Uji galeri lokal',{exact:true}))
    const catalog=await (await adminContext.request.get(base+'/api/v1/admin/catalog')).json()
    assert.equal(catalog.gallery.length,1)
    const auth=await (await adminContext.request.get(base+'/api/v1/auth/session')).json()
    for(let i=1;i<3;i++) {
      const saved=await adminContext.request.post(base+'/api/v1/admin/gallery',{
        headers:{'X-CSRF-Token':auth.csrfToken},
        data:{...catalog.gallery[0],id:'',title:'Uji slider '+i,sortOrder:i},
      })
      assert.equal(saved.status(),200)
    }
    steps.push('Admin image upload and publication persist through real browser forms')

    const publicPage=await customer.newPage();watch(publicPage)
    await publicPage.goto(base+'/#galeri')
    await has(publicPage.getByRole('heading',{name:'Hasil dari kursi kami.'}))
    await publicPage.waitForFunction(()=>document.querySelectorAll('.gallery-card img').length===3)
    await publicPage.waitForFunction(()=>[...document.querySelectorAll('.gallery-card img')].every((i)=>i.complete && i.naturalWidth>0))
    await has(publicPage.getByText('1 penilaian dari kunjungan selesai'))
    assert.equal(await publicPage.getByText('Pengunjung Uji',{exact:true}).count(),1)
    if(output) await publicPage.screenshot({path:path.join(output,'public-desktop.png'),fullPage:true})
    await publicPage.setViewportSize({width:390,height:844})
    await publicPage.locator('#galeri').scrollIntoViewIfNeeded()
    await publicPage.getByRole('button',{name:'Foto berikutnya',exact:true}).click()
    await publicPage.waitForFunction(()=>document.querySelector('.gallery-track').scrollLeft>20)
    assert(await publicPage.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth+1),'Mobile horizontal overflow')
    await publicPage.getByRole('button',{name:'Buka menu',exact:true}).click()
    await has(publicPage.getByRole('navigation',{name:'Navigasi ponsel'}))
    await publicPage.getByRole('button',{name:'Tutup menu',exact:true}).click()
    if(output) await publicPage.screenshot({path:path.join(output,'public-mobile.png'),fullPage:true})
    steps.push('Desktop and 390px public layout, mobile gallery controls, navigation and rating summary')
    assert.deepEqual(failures,[],'Unexpected server failures')
    assert.deepEqual(errors,[],'Browser runtime or CSP errors')
    console.log(JSON.stringify({passed:steps,browser:'Chromium',tls:'self-signed local test only',screenshots:output?'internal artifacts only':'disabled'}))
  } catch(error) {
    const diagnostics=[]
    for(const context of browser.contexts()) for(const page of context.pages()) {
      diagnostics.push({path:new URL(page.url()).pathname,headings:await page.locator('h1,h2').allTextContents(),alerts:await page.getByRole('alert').allTextContents(),buttons:await page.getByRole('button').allTextContents()})
      if(output) await page.screenshot({path:path.join(output,'failure-'+diagnostics.length+'.png'),fullPage:true}).catch(()=>{})
    }
    console.error(JSON.stringify({diagnostics,errors,failures}))
    throw error
  } finally {await browser.close()}
}
main().catch((error)=>{console.error(error.stack);process.exitCode=1})
