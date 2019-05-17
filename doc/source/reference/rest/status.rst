.. _rest-status

#################
Statut du service
#################

Afficher le statut du service
=============================

.. http:get:: /api/status

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Le service est actif
   :statuscode 401: Authentification d'utilisateur invalide

   **Exemples**

   * Requête :

   .. sourcecode:: http

      GET /log HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   * Réponse :

   .. sourcecode:: http

      HTTP/1.1 200 OK