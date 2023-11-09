Supprimer une instance cloud
============================

.. http:delete:: /api/clouds/(string:name)

   Supprime l'instance cloud demandée.

   :reqheader Authorization: Les identifiants de l'utilisateur REST

   :statuscode 204: L'instance cloud a été supprimée avec succès
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: L'instance cloud demandée n'existe pas

   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/clouds/aws HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT