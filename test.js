(function() {
  const debounce = require('github/debounce').default
  const {fetchText} = require('github/fetch')
  const {observe} = require('github/observe')
  const $ = require('github/jquery').default
  const {activate, deactivate} = require('github/menu')

  function onSelection() {
    const popover = document.querySelector('.js-tagsearch-popover')
    const selection = document.getSelection()
    if (selection.isCollapsed) {
      $(popover).hide()
      deactivate(popover)
      return
    }

    const text = selection.toString().trim()
    if (!text) return
    if (text.match(/\n|\s/)) return
    if (text.match(/[();&.=",]/)) return

    const file = selection.anchorNode.parentNode.closest('.js-file-line-container')
    if (!file) return

    const rect = selection.getRangeAt(0).getClientRects()[0]

    popover.style.position = 'absolute',
    popover.style.top = window.scrollY + rect.bottom + 6 + 'px'
    popover.style.left = window.scrollX + rect.left + rect.width / 2 + 'px'

    popover.querySelector('.js-tagsearch-popover-content').innerHTML = `
      <center><img src="/images/spinners/octocat-spinner-32.gif" alt="" class="loading-spinner" width="16" height="16"></center>
    `
    $(popover).show()
    activate(popover)

    const url = new URL(popover.getAttribute('data-tagsearch-url'), window.location.origin)
    const params = new URLSearchParams()
    params.set('q', text)
    params.set('context', popover.getAttribute('data-tagsearch-context'))
    url.search = params.toString()

    fetchSearchResults(url, popover)
  }

  function fetchSearchResults(url, popover, timeout) {
    if (!timeout) timeout = 2000

    fetchText(url.toString()).then(function(html) {
      if (popover.style.display === "none") return

      popover.querySelector('.js-tagsearch-popover-content').innerHTML = html

      if (popover.querySelector('.js-tagsearch-results-indexing')) {
        setTimeout(function() {
          fetchSearchResults(url, popover, timeout * 2)
        }, timeout)
      }
    })
  }

  const selected = debounce(onSelection, 200)

  observe('.js-tagsearch-popover', {
    add: function() {
      document.addEventListener('selectionchange', selected)
    },
    remove: function() {
      document.removeEventListener('selectionchange', selected)
    }
  })
})()
