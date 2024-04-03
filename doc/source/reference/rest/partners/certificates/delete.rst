[OBSOLÈTE] Supprimer un certificat
==================================

.. http:delete:: /api/partners/(string:partner)/certificates/(string:cert_name)

   Supprime le certificat demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le certificat a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire ou le certificat demandés n'existent pas


   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/partners/waarp_r66/certificates/certificat_waarp HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT