Supprimer un template d'email
=============================

.. http:delete:: /api/email/templates/(string:name)

   Supprime le template d'email demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur.

   :statuscode 204: Le template a été supprimé avec succès.
   :statuscode 401: Authentification d'utilisateur invalide.
   :statuscode 404: Le template demandé n'existe pas.


   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/email/templates/alerte_erreur HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO CONTENT
