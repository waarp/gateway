Supprimer un certificat
=======================

.. http:delete:: /api/servers/(string:server)/certificates/(string:cert_name)

   Supprime le certificat demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le certificat a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur ou le certificat demandés n'existent pas


   .. admonition:: Exemple de requête

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/servers/serveur_sftp/certificates/certificat_sftp HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   .. admonition:: Exemple de réponse

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT