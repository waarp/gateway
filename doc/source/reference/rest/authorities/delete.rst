Supprimer une autorité
======================

.. http:delete:: /api/authorities/(string:authority_name)

   Supprime l'autorité demandée.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: L'autorité a été supprimée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'autorité demandée n'existe pas

   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/authorities/tls_ca HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT