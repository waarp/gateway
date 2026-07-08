Supprimer un identifiant SMTP
==============================

.. http:delete:: /api/email/credentials/(string:emailAddress)

   Supprime l'identifiant SMTP demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur REST

   :statuscode 204: L'identifiant SMTP a été supprimé avec succès
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: L'identifiant SMTP demandé n'existe pas

   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/email/credentials/gateway@example.com HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT
