;(() => {
  function scrollIntoView(event) {
    event.target.scrollIntoView({
      behavior: 'instant',
      block: 'nearest',
      inline: 'nearest'
    })
  }

  function highlightCategory() {
    if (!window.IntersectionObserver) return
    const buttons = document.querySelectorAll('.js-category')
    const visiblity = {}
    for (const button of buttons) {
      visiblity[button.href.split('#')[1]] = 0.0
    }

    const observer = new IntersectionObserver(
      function(entries) {
        let moreVisible = null
        for (entry of entries) {
          if (entry.isIntersecting) {
            const prevVisible = visiblity[entry.target.id]
            if (entry.intersectionRatio > prevVisible) {
              // more visible
              moreVisible = entry
            } else {
              // less visible
            }
            visiblity[entry.target.id] = entry.intersectionRatio
          }
        }
        if (!moreVisible) return
        const button = document.querySelector(
          `.js-category[href="#${moreVisible.target.id}"]`
        )
        if (!button) return
        for (const el of buttons) el.classList.remove('selected')
        button.classList.add('selected')
      },
      {threshold: 0.1}
    )

    for (const category of document.querySelectorAll('.js-category-group')) {
      observer.observe(category)
    }
  }

  highlightCategory()

  function toggleSelector(elementId) {
    const style = document.getElementById(elementId).style
    style.display = style.display === 'none' ? '' : 'none'
  }

  // ESC key for hiding skin tone selector
  document.addEventListener('keyup', function(event) {
    if (event.key === 'Escape') {
      document.getElementById('skinToneSelector').style.display = 'none'
    }
  })

  for (const category of document.querySelectorAll('.js-category')) {
    category.addEventListener('click', scrollIntoView)
  }

  const emojis = document.getElementById('emojis')
  emojis.addEventListener('click', function(event) {
    const button = event.target.closest('.emoji')
    if (!button) return

    // HEREBEDRAGONS:
    // using a template literal with an EMPTY SPACE here to temporarily resolve weird issue
    // in github/github where comment box appends weird characters after the emoji insert
    parent.window.postMessage(
      {markdown: `${button.value} `, minimize: true},
      '*'
    )
  })

  const search = document.getElementById('search')

  function toggleCategoryVisibility(action) {
    for (const category of document.querySelectorAll('.emoji-category')) {
      if (action === 'hide') {
        category.classList.add('hide')
      } else if (action === 'show') {
        category.classList.remove('hide')
      }
    }
  }

  function performSearch() {
    // grab search term
    const searchTerm = search.value.toLowerCase()

    // hide emoji categories when searching
    if (searchTerm !== '') {
      toggleCategoryVisibility('hide')
    } else {
      toggleCategoryVisibility('show')
    }

    for (const emoji of document.querySelectorAll('.emoji')) {
      const emojiName = emoji.getAttribute('title')

      // show emoji if it matches search term, otherwise hide it
      if (emojiName.indexOf(searchTerm) !== -1) {
        emoji.classList.remove('hide')
      } else {
        emoji.classList.add('hide')
      }
    }
  }

  search.addEventListener('keyup', () => {
    // only run the search code once every 500ms
    const DEBOUNCE_TIME = 500

    if (window.debounceTimeout) {
      // clear the timeout so that the search function does not run if the DEBOUNCE_TIME hasn't elapsed
      clearTimeout(window.debounceTimeout)
    }

    // queue up the performSearch function to run after DEBOUNCE_TIME has elapsed
    window.debounceTimeout = setTimeout(performSearch, DEBOUNCE_TIME)
  })

  function updateSelectedSkinTone(event) {
    const selectedSkinTone = document.getElementById('selectedSkinTone')
    selectedSkinTone.innerHTML = event.target.innerHTML

    toggleSelector('skinToneSelector')
  }

  async function updateSkinTones(event) {
    const skinToneIndex = event.target.getAttribute('data-tone')
    const url = `/fetch-skin-tones?skintone=skin_tone_${skinToneIndex}`
    const response = await fetch(url, {credentials: 'same-origin'})

    if (response.ok) {
      const emojiDiv = document.getElementById('emojis')
      const html = await response.text()

      emojiDiv.innerHTML = html

      // update selected skin tone
      updateSelectedSkinTone(event)
    } else {
      console.log(response.status)
    }
  }

  const skinToneBox = document.getElementById('skinToneSelector')
  skinToneBox.addEventListener('click', updateSkinTones)

  const selectedSkinTone = document.getElementById('selectedSkinTone')
  selectedSkinTone.addEventListener('click', function() {
    toggleSelector('skinToneSelector')
  })
})()
