Supprimer un partenaire
=======================

.. http:delete:: /api/partners/(string:partner_name)

   Supprime le partenaire demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: Le partenaire a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas


   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/partners/waarp_sftp HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO CONTENT
