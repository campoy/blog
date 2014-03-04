* Design

This application, corresponding to the final exam for the Google Cloud
certification program, can be accessed on http://campoyblog.appspot.com.

** Data model

There's three kind of objects in the datastore:

- User
- Post
- Comment

Both posts and comments have a user as an ancestor, which it's the author of
the text. This gives strong consistency for user queries when he accesses his
posts and comments.

In addition to having this information in the datastore keys, there's also the
same information as a property of both Post and Comment. They both have a
Author property, containing the encoded key of a User.

Finally, comments have a PostKey field with the key of their corresponding post. 

** Application design

The application queries are done in two steps which are actually executed
concurrently.

The first step queries the 10 newest posts in the datastore. This query is
eventually consistent.

The second step queries the 10 newest posts for the current user. This query
is strongly consistent.

Then we merge both lists of posts, removing duplicates and keeping the sorting.

Once we have this list of posts we fetch the comments for all the posts
concurrently.

Fetching the comments for a posts is again done in two steps, one for all the
comments in the post, another for all the comments on the post authored by the
current user.

At the end of this we obtain the list of N newest posts with all its comments,
and we ensure that any interaction of the current user is displayed to him.

** Performance

The datastore that we have developed allows high performance limiting the
entities in a given entity group. Entity groups handle the interactions of
a given user, meaning that a user won't be able to comment or post more than
once a second. This seems a reasonable limit.

On the other side, as many comments as needed can be done in a given post,
and the list will be eventually consistent.

Finally, the usage of memcache to keep the list of the latest 10 posts and
its comments limits the reads on the datastore.


