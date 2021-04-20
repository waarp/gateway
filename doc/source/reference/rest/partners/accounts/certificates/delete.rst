Supprimer un certificat
=======================

.. http:delete:: /api/partners/(string:partner)/accounts/(string:login)/certificates/(string:cert_name)

   Supprime le certificat demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le certificat a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire, le compte ou le certificat demandés n'existent pas


   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/partners/waarp_r66/accounts/titi/certificates/certificat_titi HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT