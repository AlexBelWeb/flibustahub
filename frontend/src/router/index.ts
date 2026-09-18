import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '@/views/HomeView.vue'
import BooksView from '@/views/BooksView.vue'
import BookView from '@/views/BookView.vue'
import AuthorsView from '@/views/AuthorsView.vue'
import AuthorView from '@/views/AuthorView.vue'
import SeriesView from '@/views/SeriesView.vue'
import SeriesDetailView from '@/views/SeriesDetailView.vue'
import GenresView from '@/views/GenresView.vue'
import GenreView from '@/views/GenreView.vue'
import RecommendationsView from '@/views/RecommendationsView.vue'
import SettingsView from '@/views/SettingsView.vue'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/books', name: 'books', component: BooksView },
    { path: '/books/:workId', name: 'book', component: BookView },
    { path: '/authors', name: 'authors', component: AuthorsView },
    { path: '/authors/:authorId', name: 'author', component: AuthorView },
    { path: '/series', name: 'series', component: SeriesView },
    { path: '/series/:seriesId', name: 'seriesDetail', component: SeriesDetailView },
    { path: '/genres', name: 'genres', component: GenresView },
    { path: '/genres/:genreId', name: 'genre', component: GenreView },
    { path: '/recommendations', name: 'recommendations', component: RecommendationsView },
    { path: '/settings/:section?', name: 'settings', component: SettingsView },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})
