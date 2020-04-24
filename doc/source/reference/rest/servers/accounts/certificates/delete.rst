Supprimer un certificat
=======================

.. http:delete:: /api/servers/(string:server)/accounts/(string:login)/certificates/(string:cert_name)

   Supprime le certificat demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le certificat a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur, le compte ou le certificat demandés n'existent pas


   .. admonition:: Exemple de requête

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/servers/serveur_sftp/accounts/toto/certificates/certificat_toto HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   .. admonition:: Exemple de réponse

      .. code-block:: http

         HTTP/1.1 204 NO CONTENT