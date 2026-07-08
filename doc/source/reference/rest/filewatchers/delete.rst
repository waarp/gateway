Supprimer un filewatcher
========================

.. http:delete:: /api/filewatchers/(string:flow)

   Supprime le *filewatcher* demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur REST

   :statuscode 204: Le *filewatcher* a été supprimé avec succès
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: Le *filewatcher* demandé n'existe pas

   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/filewatchers/my-filewatcher HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT
